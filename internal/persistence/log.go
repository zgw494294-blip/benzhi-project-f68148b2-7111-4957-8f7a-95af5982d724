package persistence

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
)

// appendEvent appends a marshalled event followed by a newline to the log,
// fsyncs the file, and returns the byte offset in the file at which the
// new event line begins. Callers may pass that offset to truncateLastEvent
// to undo the append on a higher-level failure.
func appendEvent(path string, event Event) (int64, error) {
	data, err := json.Marshal(event)
	if err != nil {
		return 0, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return 0, err
	}
	offset := info.Size()
	if _, err := file.Write(append(data, '\n')); err != nil {
		return 0, err
	}
	if err := file.Sync(); err != nil {
		return 0, err
	}
	return offset, nil
}

// truncateLastEvent truncates the log file to end exactly at offset and
// fsyncs the result. It is used to roll back the most recently appended
// event when a subsequent operation fails, so the durable log no longer
// references state that was not committed via the projection snapshot.
func truncateLastEvent(path string, offset int64) error {
	if offset < 0 {
		return fmt.Errorf("截断偏移非法：%d", offset)
	}
	file, err := os.OpenFile(path, os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := file.Truncate(offset); err != nil {
		return err
	}
	return file.Sync()
}

func readEvents(path string) ([]Event, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(data) > 0 && data[len(data)-1] != '\n' {
		return nil, fmt.Errorf("事件日志末尾被截断")
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	var events []Event
	for scanner.Scan() {
		if len(bytes.TrimSpace(scanner.Bytes())) == 0 {
			return nil, fmt.Errorf("事件日志包含空记录")
		}
		var event Event
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			return nil, fmt.Errorf("事件日志 JSON 损坏: %w", err)
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return events, nil
}
