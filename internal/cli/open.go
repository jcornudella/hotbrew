package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/jcornudella/hotbrew/internal/store"
)

// Open handles `hotbrew open <id>`.
// Opens the item URL in the default browser and marks it as read.
func Open(st *store.Store, idPrefix string) {
	if idPrefix == "" {
		fmt.Println("Usage: hotbrew open <id-prefix>")
		fmt.Println("\nUse 'hotbrew list' to see item IDs.")
		os.Exit(1)
	}

	item, err := st.GetItem(idPrefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Item not found: %v\n", err)
		os.Exit(1)
	}

	if item.URL == "" {
		fmt.Println("This item has no URL.")
		return
	}

	// Open in browser
	openBrowser(item.URL)
	fmt.Printf("✓ Opened: %s\n", item.Title)

	_ = st.RecordFeedbackEvent(store.FeedbackActionOpen, item.ID, "")

	// Mark as read
	if err := st.MarkRead(item.ID); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not mark as read: %v\n", err)
	} else {
		_ = st.RecordFeedbackEvent(store.FeedbackActionRead, item.ID, "")
	}
}

// Save handles `hotbrew save <id>`.
func Save(st *store.Store, idPrefix string) {
	if idPrefix == "" {
		fmt.Println("Usage: hotbrew save <id-prefix>")
		os.Exit(1)
	}

	item, err := st.GetItem(idPrefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Item not found: %v\n", err)
		os.Exit(1)
	}

	if err := st.MarkSaved(item.ID); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving item: %v\n", err)
		os.Exit(1)
	}
	_ = st.RecordFeedbackEvent(store.FeedbackActionSave, item.ID, "")

	fmt.Printf("★ Saved: %s\n", item.Title)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		fmt.Printf("Open: %s\n", url)
		return
	}
	_ = cmd.Start()
}
