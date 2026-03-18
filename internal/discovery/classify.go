package discovery

import "sync"

// Session represents a fully classified Claude Code session ready for display.
type Session struct {
	PID           int
	PPID          int
	CPU           float64
	Mem           float64
	Elapsed       string
	Command       string
	WorkDir       string
	Status        string // "working" | "waiting" | "idle"
	Source        string // "VSCode" | "CLI" | ...
	SubagentCount int
}

// RawProc holds the raw data from a process listing before classification.
type RawProc struct {
	PID     int
	PPID    int
	CPU     float64
	Mem     float64
	Elapsed string
	Command string
}

// DetermineStatus classifies a session based on CPU usage and JSONL state.
//   - CPU above threshold → working (always wins)
//   - assistant + tool_use → waiting (Claude is showing a tool-approval prompt)
//   - assistant + end_turn → idle (Claude finished responding; user may or may not be present)
//   - user → idle (if Claude were processing, CPU would be high)
//   - anything else → idle
func DetermineStatus(cpu, cpuThreshold float64, state SessionState) string {
	if cpu > cpuThreshold {
		return "working"
	}
	if state.Type == "assistant" && state.StopReason == "tool_use" {
		return "waiting"
	}
	return "idle"
}

// ClassifySessions takes raw process data and enriches each process with
// working directory, source IDE, and status. Slow I/O (lsof, ps, file reads)
// is parallelised across sessions.
func ClassifySessions(procs []RawProc, cpuThreshold float64) []Session {
	claudePIDs := make(map[int]bool, len(procs))
	for _, p := range procs {
		claudePIDs[p.PID] = true
	}

	subagentCount := make(map[int]int)
	var mainProcs []RawProc

	for _, p := range procs {
		if claudePIDs[p.PPID] {
			subagentCount[p.PPID]++
		} else {
			mainProcs = append(mainProcs, p)
		}
	}

	// Pre-allocate result slice so goroutines can write by index without locks
	sessions := make([]Session, len(mainProcs))

	var wg sync.WaitGroup
	for i, p := range mainProcs {
		wg.Add(1)
		go func(idx int, p RawProc) {
			defer wg.Done()

			s := Session{
				PID:           p.PID,
				PPID:          p.PPID,
				CPU:           p.CPU,
				Mem:           p.Mem,
				Elapsed:       p.Elapsed,
				Command:       p.Command,
				SubagentCount: subagentCount[p.PID],
			}

			// These are the slow calls — lsof/proc, ps, file I/O — now parallel
			s.WorkDir = GetWorkDir(p.PID)
			s.Source = GetSource(p.PPID, p.Command)

			// Determine status from CPU + JSONL state
			sessionID := getSessionID(p.Command, s.WorkDir)
			jsonlPath := getSessionFilePath(sessionID, s.WorkDir)
			state := readSessionState(jsonlPath)
			s.Status = DetermineStatus(p.CPU, cpuThreshold, state)

			sessions[idx] = s
		}(i, p)
	}
	wg.Wait()

	return sessions
}
