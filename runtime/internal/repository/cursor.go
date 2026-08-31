package repository

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

type Cursor struct {
	CreatedAt time.Time
	ID        string
}

type cursorWire struct {
	Time string `json:"t"`
	ID   string `json:"i"`
}

func EncodeCursor(createdAt time.Time, id string) string {
	encoded, _ := json.Marshal(cursorWire{Time: createdAt.UTC().Format(time.RFC3339Nano), ID: id})
	return base64.RawURLEncoding.EncodeToString(encoded)
}

func DecodeCursor(value string) (Cursor, error) {
	if value == "" {
		return Cursor{}, fmt.Errorf("cursor is empty")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return Cursor{}, fmt.Errorf("decode cursor: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(decoded))
	decoder.DisallowUnknownFields()
	var wire cursorWire
	if err := decoder.Decode(&wire); err != nil {
		return Cursor{}, fmt.Errorf("decode cursor value: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Cursor{}, fmt.Errorf("decode cursor value: trailing content")
	}
	createdAt, err := time.Parse(time.RFC3339Nano, wire.Time)
	if err != nil || wire.Time != createdAt.UTC().Format(time.RFC3339Nano) {
		return Cursor{}, fmt.Errorf("decode cursor timestamp")
	}
	if strings.TrimSpace(wire.ID) == "" || wire.ID != strings.TrimSpace(wire.ID) {
		return Cursor{}, fmt.Errorf("decode cursor id")
	}
	return Cursor{CreatedAt: createdAt, ID: wire.ID}, nil
}
