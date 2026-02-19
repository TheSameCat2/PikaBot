package rcon

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	packetTypeResponseValue = 0
	packetTypeExecCommand   = 2
	packetTypeAuth          = 3
)

type Client struct {
	host    string
	port    int
	pass    string
	timeout time.Duration
}

func New(host string, port int, pass string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &Client{host: host, port: port, pass: pass, timeout: timeout}
}

func (c *Client) ShowPlayers(ctx context.Context) ([]string, error) {
	response, err := c.execute(ctx, "ShowPlayers")
	if err != nil {
		return nil, err
	}
	return ParseShowPlayers(response), nil
}

func (c *Client) execute(ctx context.Context, command string) (string, error) {
	addr := net.JoinHostPort(c.host, strconv.Itoa(c.port))
	dialer := &net.Dialer{Timeout: c.timeout}
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return "", fmt.Errorf("dial rcon %s: %w", addr, err)
	}
	defer conn.Close()

	deadline := time.Now().Add(c.timeout)
	if dl, ok := ctx.Deadline(); ok && dl.Before(deadline) {
		deadline = dl
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return "", fmt.Errorf("set deadline: %w", err)
	}

	if err := writePacket(conn, 1, packetTypeAuth, c.pass); err != nil {
		return "", fmt.Errorf("send auth packet: %w", err)
	}
	if err := readAuthResponse(conn); err != nil {
		return "", err
	}

	if err := writePacket(conn, 2, packetTypeExecCommand, command); err != nil {
		return "", fmt.Errorf("send command packet: %w", err)
	}

	first, err := readPacket(conn)
	if err != nil {
		return "", fmt.Errorf("read command response: %w", err)
	}
	if first.ID == -1 {
		return "", errors.New("rcon command rejected")
	}

	builder := strings.Builder{}
	if first.Type == packetTypeResponseValue {
		builder.WriteString(first.Body)
	}

	_ = conn.SetReadDeadline(time.Now().Add(150 * time.Millisecond))
	for {
		next, err := readPacket(conn)
		if err != nil {
			if isTimeout(err) {
				break
			}
			return "", fmt.Errorf("read additional response packet: %w", err)
		}
		if next.Type == packetTypeResponseValue {
			if builder.Len() > 0 && !strings.HasSuffix(builder.String(), "\n") {
				builder.WriteString("\n")
			}
			builder.WriteString(next.Body)
		}
	}

	return builder.String(), nil
}

func readAuthResponse(conn net.Conn) error {
	for i := 0; i < 3; i++ {
		pkt, err := readPacket(conn)
		if err != nil {
			return fmt.Errorf("read auth response: %w", err)
		}
		if pkt.Type != packetTypeExecCommand {
			continue
		}
		if pkt.ID == -1 {
			return errors.New("rcon auth failed")
		}
		return nil
	}
	return errors.New("rcon auth failed: no auth response packet")
}

type packet struct {
	ID   int32
	Type int32
	Body string
}

func writePacket(w io.Writer, id int32, typ int32, body string) error {
	payloadLen := 4 + 4 + len(body) + 2
	buf := make([]byte, 4+payloadLen)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(payloadLen))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(id))
	binary.LittleEndian.PutUint32(buf[8:12], uint32(typ))
	copy(buf[12:12+len(body)], []byte(body))
	buf[len(buf)-2] = 0
	buf[len(buf)-1] = 0
	_, err := w.Write(buf)
	if err != nil {
		return fmt.Errorf("write packet: %w", err)
	}
	return nil
}

func readPacket(r io.Reader) (packet, error) {
	var length int32
	if err := binary.Read(r, binary.LittleEndian, &length); err != nil {
		return packet{}, err
	}
	if length < 10 || length > 4096 {
		return packet{}, fmt.Errorf("invalid packet length %d", length)
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return packet{}, err
	}
	id := int32(binary.LittleEndian.Uint32(buf[0:4]))
	typ := int32(binary.LittleEndian.Uint32(buf[4:8]))
	body := strings.TrimRight(string(buf[8:len(buf)-2]), "\x00")
	return packet{ID: id, Type: typ, Body: body}, nil
}

func isTimeout(err error) bool {
	netErr := net.Error(nil)
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	return false
}
