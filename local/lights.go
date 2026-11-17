package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func pushLights(kind, openHost, wledHost string, r, g, b int) string {
	kind = strings.ToLower(strings.TrimSpace(kind))
	var notes []string
	if kind == "keyboard" || kind == "both" {
		if err := pushOpenRGB(openHost, r, g, b); err != nil {
			notes = append(notes, "keyboard: "+err.Error())
		} else {
			notes = append(notes, "keyboard lit")
		}
	}
	if kind == "room" || kind == "both" {
		if err := pushWLED(wledHost, r, g, b); err != nil {
			notes = append(notes, "room: "+err.Error())
		} else {
			notes = append(notes, "room lit")
		}
	}
	if len(notes) == 0 {
		return "lights off"
	}
	return strings.Join(notes, " · ")
}

func pushWLED(host string, r, g, b int) error {
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("no WLED address")
	}
	if !strings.Contains(host, "://") {
		host = "http://" + host
	}
	parsed, err := url.Parse(host)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("bad WLED address")
	}
	host = strings.TrimRight(parsed.String(), "/")
	body, _ := json.Marshal(map[string]any{
		"on": true, "bri": 180,
		"seg": []any{map[string]any{"id": 0, "col": []any{[]int{r, g, b}}}},
	})
	req, err := http.NewRequest(http.MethodPost, host+"/json/state", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 1200 * time.Millisecond}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("WLED status %d", res.StatusCode)
	}
	return nil
}

func pushOpenRGB(host string, r, g, b int) error {
	host = strings.TrimSpace(host)
	if host == "" {
		host = "127.0.0.1"
	}
	if !strings.Contains(host, ":") {
		host += ":6742"
	}
	conn, err := net.DialTimeout("tcp", host, 700*time.Millisecond)
	if err != nil {
		return fmt.Errorf("OpenRGB is not running")
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2500 * time.Millisecond))
	ver := make([]byte, 4)
	binary.LittleEndian.PutUint32(ver, 4)
	if err := writeORGB(conn, 0, 40, ver); err != nil {
		return err
	}
	if _, _, err := readORGB(conn); err != nil {
		return fmt.Errorf("OpenRGB did not answer")
	}
	if err := writeORGB(conn, 0, 50, []byte("Valvet\x00")); err != nil {
		return err
	}
	if err := writeORGB(conn, 0, 0, nil); err != nil {
		return err
	}
	_, data, err := readORGB(conn)
	if err != nil {
		return err
	}
	if len(data) < 4 {
		return fmt.Errorf("no keyboard reply")
	}
	count := int(binary.LittleEndian.Uint32(data[:4]))
	if count <= 0 || count > 32 {
		return fmt.Errorf("no RGB devices")
	}
	color := uint32(b)<<16 | uint32(g)<<8 | uint32(r)
	sizes := []int{1, 5, 16, 20, 64, 88, 104, 126}
	for dev := 0; dev < count; dev++ {
		_ = writeORGB(conn, uint32(dev), 1100, nil)
		for _, n := range sizes {
			buf := make([]byte, 4+n*4)
			binary.LittleEndian.PutUint32(buf[0:4], uint32(n))
			for i := 0; i < n; i++ {
				binary.LittleEndian.PutUint32(buf[4+i*4:], color)
			}
			if err := writeORGB(conn, uint32(dev), 1050, buf); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeORGB(conn net.Conn, dev, id uint32, data []byte) error {
	header := make([]byte, 16)
	copy(header[0:4], "ORGB")
	binary.LittleEndian.PutUint32(header[4:8], dev)
	binary.LittleEndian.PutUint32(header[8:12], id)
	binary.LittleEndian.PutUint32(header[12:16], uint32(len(data)))
	if _, err := conn.Write(header); err != nil {
		return err
	}
	if len(data) > 0 {
		_, err := conn.Write(data)
		return err
	}
	return nil
}

func readORGB(conn net.Conn) (uint32, []byte, error) {
	header := make([]byte, 16)
	if _, err := ioReadFull(conn, header); err != nil {
		return 0, nil, err
	}
	if string(header[0:4]) != "ORGB" {
		return 0, nil, fmt.Errorf("not OpenRGB")
	}
	id := binary.LittleEndian.Uint32(header[8:12])
	size := binary.LittleEndian.Uint32(header[12:16])
	if size > 1<<20 {
		return 0, nil, fmt.Errorf("packet too big")
	}
	data := make([]byte, size)
	if size > 0 {
		if _, err := ioReadFull(conn, data); err != nil {
			return 0, nil, err
		}
	}
	return id, data, nil
}

func ioReadFull(conn net.Conn, buf []byte) (int, error) {
	got := 0
	for got < len(buf) {
		n, err := conn.Read(buf[got:])
		got += n
		if err != nil {
			return got, err
		}
	}
	return got, nil
}
