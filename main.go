package main

import (
	"fmt"
	"os"

	"strings"

	"os/user"

	"strconv"

	"bytes"
	"os/exec"

	"github.com/fatih/color"
)

var (
	CHR_RIGHT_ARROW   = "\ue0b0"
	CHR_PLUS_MINUS    = "\u00b1"
	CHR_GIT_BRANCH    = "\ue0a0"
	CHR_GIT_DETATCHED = "\u27a6"
	CHR_FAILURE       = "\u2718"
	CHR_LIGHTNING     = "\u26a1"
	CHR_COG           = "\u2699"
	CHR_DOWN_ARROW    = "\u2193"
	CHR_UP_ARROW      = "\u2191"
	CHR_SUCCESS       = "\u2714"
)

type InfoBlock struct {
	Foreground color.Attribute
	Background color.Attribute
	Bold       bool
	Underline  bool
	Text       string
}

func (ib *InfoBlock) Print() {
	c := color.New(ib.Foreground).Add(ib.Background)
	if ib.Bold {
		c.Add(color.Bold)
	} else if ib.Underline {
		c.Add(color.Underline)
	}
	c.Print(" " + ib.Text + " ")
}

func makeBlock(fg color.Attribute, bg color.Attribute, text string) *InfoBlock {
	return &InfoBlock{
		Foreground: fg,
		Background: bg,
		Bold:       true,
		Underline:  false,
		Text:       text,
	}
}

func printBlockList(ls []*InfoBlock) {
	for i, ib := range ls {
		if i != 0 {
			// Print the ending arrow of the previous one
			tc := color.New(ib.Background).Add(ls[i-1].Background - 10)
			tc.Print(CHR_RIGHT_ARROW)
		}
		ib.Print()
		if i == len(ls)-1 {
			// Last item, print a final arrow
			tc := color.New(ib.Background - 10)
			tc.Print(CHR_RIGHT_ARROW)
		}
	}
}

func getCwd() string {
	pwd, err := os.Getwd()
	if err != nil {
		return "!!ERR"
	}

	homedir := os.Getenv("HOME")

	if strings.HasPrefix(pwd, homedir) {
		pwd = "~" + pwd[len(homedir):]
	}

	return pwd
}

func makeRootBlock() *InfoBlock {
	return nil
}

func makeOtherUserBlock(usr *user.User) *InfoBlock {
	hostname, _ := os.Hostname()

	block := makeBlock(color.FgHiYellow, color.BgBlack, usr.Username+"@"+hostname)

	uid, _ := strconv.Atoi(usr.Uid)

	if uid == 0 {
		// we are root
		block.Text = CHR_LIGHTNING + " " + block.Text
	}

	return block
}

func gitBlock() *InfoBlock {
	cmd := exec.Command("git", "status", "--porcelain=v2", "--ignore-submodules", "--branch")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil
	}

	gitBlock := makeBlock(color.FgHiBlack, color.BgHiGreen, "")

	gitStatus := out.String()
	statusLines := strings.Split(gitStatus, "\n")

	gitHeaders := make(map[string]string)

	isDirty := false

	// Parse porcelain v2
	for _, v := range statusLines {
		if strings.HasPrefix(v, "#") {
			parts := strings.SplitN(v, " ", 3)
			if len(parts) == 3 {
				gitHeaders[parts[1]] = parts[2]
			}
		} else {
			if len(v) > 0 {
				isDirty = true
				break
			}
		}
	}

	// Get our current HEAD (detached, branch, etc)
	if head, ok := gitHeaders["branch.head"]; ok {
		detached := false

		if head == "(detached)" {
			rev, ok := gitHeaders["branch.oid"]
			detached = true
			if ok {
				head = rev[:8]
			}
		}

		if detached {
			gitBlock.Text = CHR_GIT_DETATCHED + " " + head
		} else {
			gitBlock.Text = CHR_GIT_BRANCH + " " + head
		}
	}

	// Change output if repo is dirty
	if isDirty {
		gitBlock.Text += CHR_PLUS_MINUS
		gitBlock.Background = color.BgHiYellow
	}

	// Check if we've diverged at all from remote
	if diff, ok := gitHeaders["branch.ab"]; ok {
		// we're missing some commits
		diffp := strings.Split(diff, " ")

		diffMessage := ""

		if diffp[0] != "+0" {
			// more than 0 extra local commits
			diffMessage = CHR_UP_ARROW + diffp[0][1:]
		}
		if diffp[1] != "-0" {
			// more than 0 extra remote commits
			if len(diffMessage) > 0 {
				diffMessage += " "
			}
			diffMessage += CHR_DOWN_ARROW + diffp[1][1:]
		}

		if len(diffMessage) > 0 {
			gitBlock.Text += " " + diffMessage
		}
	}

	return gitBlock
}

func main() {
	list := make([]*InfoBlock, 0)
	color.NoColor = false // Force coloured output for PS1

	// Check for the last bash return code
	// We pass this variable to this program from Bash as there
	// is no convenient way to pass it otherwise
	if len(os.Args) > 1 {
		code := os.Args[1]
		if code != "0" {
			list = append(list, makeBlock(color.FgHiBlack, color.BgHiRed, CHR_FAILURE+" "+code))
		} else {
			list = append(list, makeBlock(color.FgHiBlack, color.BgHiGreen, CHR_SUCCESS))
		}
	}

	// Add the block for a non-default user
	currentUser, _ := user.Current()
	if currentUser.Username != os.Getenv("DEFAULT_USER") {
		list = append(list, makeOtherUserBlock(currentUser))
	}

	// Path to CWD block
	cwdB := makeBlock(color.FgHiBlack, color.BgHiBlue, getCwd())
	list = append(list, cwdB)

	// Git block
	gitB := gitBlock()
	if gitB != nil {
		list = append(list, gitB)
	}

	// Print the list
	fmt.Printf("\n\033[2K") // go to a new line and clear it for us
	printBlockList(list)

	// Reset the colour and print the 'K' ANSI control code
	// which is "Erase in Line".
	// K = clear from cursor to end
	// 1K = clear from cursor to start of line
	// 2K = clear entire line
	spaceC := color.New(color.Reset)
	spaceC.Printf(" \033[K")

	//color.Unset() // Reset back for the command input
}
