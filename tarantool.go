package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/tarantool/go-tarantool"
)

type TarantoolClient struct {
	conn *tarantool.Connection
}

func NewTarantoolClient() (*TarantoolClient, error) {
	host := os.Getenv("TARANTOOL_HOST")
	port := os.Getenv("TARANTOOL_PORT")

	opts := tarantool.Opts{
		User:          "admin",
		Pass:          "password",
		Timeout:       15 * time.Second,
		Reconnect:     2 * time.Second,
		MaxReconnects: 15,
	}

	var conn *tarantool.Connection
	var err error
	maxRetries := 7
	initialDelay := 1 * time.Second

	for i := 0; i < maxRetries; i++ {
		conn, err = tarantool.Connect(fmt.Sprintf("%s:%s", host, port), opts)
		if err == nil {
			resp, pingErr := conn.Ping()
			if pingErr == nil && resp.Code == tarantool.OkCode {
				logrus.Info("Successfully connected to Tarantool")
				return &TarantoolClient{conn: conn}, nil
			}
			if conn != nil {
				conn.Close()
			}
		}

		if i < maxRetries-1 {
			delay := time.Duration(math.Pow(2, float64(i))) * initialDelay
			logrus.Warnf("Connection attempt %d failed, retrying in %v: %v", i+1, delay, err)
			time.Sleep(delay)
		}
	}

	return nil, fmt.Errorf("tarantool connection failed after %d attempts: %w", maxRetries, err)
}

func (tc *TarantoolClient) CreateVote(id, creator, question string, options []string) error {
	if len(options) == 0 {
		return fmt.Errorf("at least one option required")
	}

	optionsMap := make(map[string]int)
	for _, opt := range options {
		if opt == "" {
			return fmt.Errorf("empty option not allowed")
		}
		optionsMap[opt] = 0
	}

	_, err := tc.conn.Insert("votes", []interface{}{
		id, creator, question, optionsMap, "active", time.Now().UTC(),
	})
	return err
}

func (tc *TarantoolClient) RecordVote(id, option string) error {
	_, err := tc.conn.Update("votes", "primary", []interface{}{id}, []interface{}{
		[]interface{}{"+", fmt.Sprintf("options[%q]", option), 1},
	})
	return err
}

func (tc *TarantoolClient) GetNextID() (string, error) {
	resp, err := tc.conn.Call("box.atomic", []interface{}{
		"function() return box.space.counters:update('vote_id', {{'+', 2, 1}}) end",
	})
	if err != nil {
		return "", fmt.Errorf("getNextID failed: %w", err)
	}

	if len(resp.Data) == 0 {
		return "", fmt.Errorf("empty response from counters space")
	}

	tuple, ok := resp.Data[0].([]interface{})
	if !ok || len(tuple) < 3 {
		return "", fmt.Errorf("invalid counter tuple format")
	}

	id, ok := tuple[2].(uint64)
	if !ok {
		return "", fmt.Errorf("invalid counter value type")
	}

	return strconv.FormatUint(id, 10), nil
}

func (tc *TarantoolClient) GetResults(id string) (map[string]int, error) {
	resp, err := tc.conn.Select("votes", "primary", 0, 1, tarantool.IterEq, []interface{}{id})
	if err != nil {
		return nil, fmt.Errorf("getResults query failed: %w", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("vote not found")
	}

	tuple, ok := resp.Data[0].([]interface{})
	if !ok || len(tuple) < 5 {
		return nil, fmt.Errorf("invalid vote tuple format")
	}

	options, ok := tuple[3].(map[string]int)
	if !ok {
		return nil, fmt.Errorf("invalid options format")
	}

	return options, nil
}

func (tc *TarantoolClient) EndVote(id string) error {
	_, err := tc.conn.Update("votes", "primary", []interface{}{id}, []interface{}{
		[]interface{}{"=", 4, "ended"},
		[]interface{}{"=", 5, time.Now().UTC()},
	})
	return err
}

func (tc *TarantoolClient) DeleteVote(id string) error {
	_, err := tc.conn.Delete("votes", "primary", []interface{}{id})
	return err
}
