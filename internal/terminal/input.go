package terminal

// Key represents a keypress event from the terminal.
type Key int

const (
	KeyNone  Key = iota
	KeyUp        // Arrow up or 'k'
	KeyDown      // Arrow down or 'j'
	KeyEnter     // Enter/Return
	KeyQuit      // 'q' or 'Q'
)
