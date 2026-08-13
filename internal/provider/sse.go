package provider

import (
	"bufio"
	"io"
	"strings"
)

// sseEvent is a single decoded server-sent event.
type sseEvent struct {
	Event string
	Data  string
}

// readSSE parses a stream of SSE frames, yielding one event per emitted block.
func readSSE(r io.Reader) <-chan sseEvent {
	out := make(chan sseEvent)
	go func() {
		defer close(out)
		reader := bufio.NewReaderSize(r, 64*1024)
		var eventName string
		var dataLines []string

		flush := func() {
			if len(dataLines) == 0 {
				eventName = ""
				return
			}
			out <- sseEvent{
				Event: eventName,
				Data:  strings.Join(dataLines, "\n"),
			}
			eventName = ""
			dataLines = nil
		}

		for {
			line, err := reader.ReadString('\n')
			line = strings.TrimRight(line, "\r\n")

			switch {
			case strings.HasPrefix(line, "event:"):
				eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			case strings.HasPrefix(line, "data:"):
				dataLines = append(dataLines, strings.TrimPrefix(line, "data:"))
			case line == "":
				flush()
			}

			if err != nil {
				flush()
				return
			}
		}
	}()
	return out
}
