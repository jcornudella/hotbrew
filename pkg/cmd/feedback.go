package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jcornudella/hotbrew/internal/config"
	"github.com/jcornudella/hotbrew/internal/store"
	"github.com/jcornudella/hotbrew/internal/store/repo"
)

func promptIssueRating() {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("How good was today's issue? (1=meh, 4=amazing)")
	fmt.Print("Rating [1-4, Enter to skip]: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return
	}
	value := strings.TrimSpace(input)
	if value == "" {
		fmt.Println("Thanks! We'll keep brewing.")
		return
	}

	rating, err := strconv.Atoi(value)
	if err != nil || rating < 1 || rating > 4 {
		fmt.Println("Got it — skipping feedback.")
		return
	}

	if err := recordRating(rating); err != nil {
		fmt.Printf("Couldn't record feedback: %v\n", err)
	} else {
		fmt.Println("Appreciate the feedback!")
	}
}

func recordRating(rating int) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	st, err := store.Open(cfg.GetDBPath())
	if err != nil {
		return err
	}
	defer st.Close()

	repo := repo.New(st)
	return repo.InsertFeedback(rating, "")
}
