package matrix

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"pikabot/internal/commands"
	"pikabot/internal/config"
	"pikabot/internal/dockerctl"
	"pikabot/internal/logx"
	"pikabot/internal/rcon"

	"maunium.net/go/mautrix"
	"maunium.net/go/mautrix/event"
	"maunium.net/go/mautrix/id"
)

type Bot struct {
	cfg      config.Config
	log      *logx.Logger
	matrix   *mautrix.Client
	docker   *dockerctl.Controller
	rcon     *rcon.Client
	roomID   id.RoomID
	busy     atomic.Bool
	allowed  map[string]struct{}
	selfUser id.UserID
}

func New(ctx context.Context, cfg config.Config, logger *logx.Logger) (*Bot, error) {
	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}

	dockerController, err := dockerctl.New(cfg.DockerContainerName)
	if err != nil {
		return nil, err
	}

	accessToken, err := resolveAccessToken(cfg)
	if err != nil {
		return nil, fmt.Errorf("resolve matrix access token: %w", err)
	}

	userID := id.UserID(strings.TrimSpace(cfg.MatrixUserID))
	matrixClient, err := mautrix.NewClient(cfg.MatrixHomeserver, userID, accessToken)
	if err != nil {
		return nil, fmt.Errorf("create matrix client: %w", err)
	}

	if matrixClient.UserID == "" {
		whoami, whoamiErr := matrixClient.Whoami(ctx)
		if whoamiErr != nil {
			return nil, fmt.Errorf("resolve MATRIX_USER_ID with /whoami: %w", whoamiErr)
		}
		matrixClient.UserID = whoami.UserID
	}

	if strings.TrimSpace(cfg.MatrixAccessToken) == "" && !tokenFileExists(cfg.AccessTokenPath()) {
		if err := writeSecretFile(cfg.AccessTokenPath(), []byte(matrixClient.AccessToken+"\n")); err != nil {
			return nil, fmt.Errorf("persist matrix access token: %w", err)
		}
	}

	syncer := mautrix.NewDefaultSyncer()
	syncer.FilterJSON = &mautrix.Filter{
		Room: &mautrix.RoomFilter{
			Rooms: []id.RoomID{id.RoomID(cfg.MatrixRoomID)},
			Timeline: &mautrix.FilterPart{
				Types: []event.Type{event.EventMessage},
			},
		},
	}

	matrixClient.Syncer = syncer
	matrixClient.Store = NewFileSyncStore(cfg.SyncTokenPath())

	bot := &Bot{
		cfg:      cfg,
		log:      logger,
		matrix:   matrixClient,
		docker:   dockerController,
		rcon:     rcon.New(cfg.RCONHost, cfg.RCONPort, cfg.RCONPass, 5*time.Second),
		roomID:   id.RoomID(cfg.MatrixRoomID),
		allowed:  cfg.AllowedMXIDs,
		selfUser: matrixClient.UserID,
	}

	syncer.OnEventType(event.EventMessage, bot.handleMessage)
	return bot, nil
}

func (b *Bot) Run(ctx context.Context) error {
	if err := b.bootstrapSyncToken(ctx); err != nil {
		return err
	}

	b.log.Info("matrix sync started", "room_id", b.roomID.String(), "user_id", b.selfUser.String())
	err := b.matrix.SyncWithContext(ctx)
	if err != nil && ctx.Err() == nil {
		return err
	}
	return nil
}

func (b *Bot) Close() error {
	return b.docker.Close()
}

func (b *Bot) bootstrapSyncToken(ctx context.Context) error {
	nextBatch, err := b.matrix.Store.LoadNextBatch(ctx, b.matrix.UserID)
	if err != nil {
		return fmt.Errorf("load sync token: %w", err)
	}
	if nextBatch != "" {
		return nil
	}

	resp, err := b.matrix.SyncRequest(ctx, 0, "", "", false, "")
	if err != nil {
		return fmt.Errorf("initial sync request for token bootstrap: %w", err)
	}
	if resp.NextBatch == "" {
		return errors.New("initial sync returned empty next_batch")
	}
	if err := b.matrix.Store.SaveNextBatch(ctx, b.matrix.UserID, resp.NextBatch); err != nil {
		return fmt.Errorf("save initial sync token: %w", err)
	}
	b.log.Info("initialized sync token without replay")
	return nil
}

func (b *Bot) handleMessage(ctx context.Context, evt *event.Event) {
	if evt == nil {
		return
	}
	if evt.RoomID != b.roomID {
		return
	}
	if evt.Sender == b.selfUser {
		return
	}
	if _, ok := b.allowed[evt.Sender.String()]; !ok {
		return
	}

	if err := evt.Content.ParseRaw(evt.Type); err != nil {
		b.log.Warn("failed parsing matrix event content", "event_id", evt.ID.String(), "err", err.Error())
		return
	}
	content := evt.Content.AsMessage()
	if content == nil || !content.MsgType.IsText() {
		return
	}

	cmd := commands.Parse(content.Body, b.cfg.CommandPrefix)
	if cmd.Type == commands.Unknown {
		return
	}

	if !b.busy.CompareAndSwap(false, true) {
		b.reply(ctx, "busy, try again")
		return
	}
	defer b.busy.Store(false)

	switch cmd.Type {
	case commands.StartPal:
		b.handleStart(ctx)
	case commands.StopPal:
		b.handleStop(ctx)
	}
}

func (b *Bot) handleStart(ctx context.Context) {
	status, err := b.docker.Status(ctx)
	if err != nil {
		b.reply(ctx, "error checking server status: "+err.Error())
		return
	}
	if !status.Exists {
		b.reply(ctx, "configured container was not found")
		return
	}
	if status.Running {
		b.reply(ctx, "server is already running")
		return
	}

	if err := b.docker.Start(ctx); err != nil {
		b.reply(ctx, "failed to start server: "+err.Error())
		return
	}
	b.reply(ctx, "starting Palworld server...")
}

func (b *Bot) handleStop(ctx context.Context) {
	status, err := b.docker.Status(ctx)
	if err != nil {
		b.reply(ctx, "error checking server status: "+err.Error())
		return
	}
	if !status.Exists {
		b.reply(ctx, "configured container was not found")
		return
	}
	if !status.Running {
		b.reply(ctx, "server is already stopped")
		return
	}

	checkCtx, cancelCheck := context.WithTimeout(ctx, 5*time.Second)
	players, err := b.rcon.ShowPlayers(checkCtx)
	cancelCheck()
	if err != nil {
		b.reply(ctx, "refused to stop: could not confirm zero players via RCON")
		b.log.Warn("rcon check failed; stop aborted", "err", err.Error())
		return
	}
	if len(players) > 0 {
		b.reply(ctx, "abort: players are online: "+strings.Join(players, ", "))
		return
	}

	stopCtx, cancelStop := context.WithTimeout(ctx, 30*time.Second)
	err = b.docker.Stop(stopCtx, 30*time.Second)
	cancelStop()
	if err != nil {
		b.reply(ctx, "failed to stop server: "+err.Error())
		return
	}

	b.reply(ctx, "server stopped")
}

func (b *Bot) reply(ctx context.Context, text string) {
	if _, err := b.matrix.SendText(ctx, b.roomID, text); err != nil {
		b.log.Error("failed sending matrix message", "err", err.Error())
	}
}

func resolveAccessToken(cfg config.Config) (string, error) {
	if token := strings.TrimSpace(cfg.MatrixAccessToken); token != "" {
		return token, nil
	}

	if fileToken := readSecretFile(cfg.AccessTokenPath()); fileToken != "" {
		return fileToken, nil
	}

	loginClient, err := mautrix.NewClient(cfg.MatrixHomeserver, id.UserID(cfg.MatrixUserID), "")
	if err != nil {
		return "", fmt.Errorf("create matrix login client: %w", err)
	}

	resp, err := loginClient.Login(context.Background(), &mautrix.ReqLogin{
		Type: mautrix.AuthTypePassword,
		Identifier: mautrix.UserIdentifier{
			Type: mautrix.IdentifierTypeUser,
			User: cfg.MatrixUser,
		},
		Password:         cfg.MatrixPassword,
		StoreCredentials: true,
	})
	if err != nil {
		return "", fmt.Errorf("matrix password login failed: %w", err)
	}

	if err := writeSecretFile(cfg.AccessTokenPath(), []byte(resp.AccessToken+"\n")); err != nil {
		return "", fmt.Errorf("persist access token: %w", err)
	}
	return resp.AccessToken, nil
}

func readSecretFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func writeSecretFile(path string, data []byte) error {
	return writeFileAtomically(path, data, 0o600)
}

func tokenFileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
