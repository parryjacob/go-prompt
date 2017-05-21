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

// Char* consts
var (
	CharRightArrow  = "\ue0b0"
	CharPlusMinus   = "\u00b1"
	CharGitBranch   = "\ue0a0"
	CharGitDetached = "\u27a6"
	CharFailure     = "\u2718"
	CharLightning   = "\u26a1"
	CharCog         = "\u2699"
	CharDownArrow   = "\u2193"
	CharUpArrow     = "\u2191"
	CharSuccess     = "\u2714"
)

// DefaultBackgroundColour represents the Bash colour number
// for the default background colour
var DefaultBackgroundColour = 49

// InfoBlock represents a single segment of the prompt
type InfoBlock struct {
	Foreground color.Attribute
	Background color.Attribute
	Bold       bool
	Underline  bool
	Text       string
}

// Print prints the segment
func (ib *InfoBlock) Print() {
	printBashColor(int(ib.Foreground), int(ib.Background), ib.Bold)
	fmt.Print(" " + ib.Text + " ")
}

func printBashColor(fg int, bg int, bold bool) {
	s := 0
	if bold {
		s = 1
	}
	fmt.Print("\001\033[" + strconv.Itoa(s) + ";" + strconv.Itoa(fg) + ";" + strconv.Itoa(bg) + "m\002")
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
			printBashColor(int(ls[i-1].Background-10), int(ib.Background), false)
			fmt.Print(CharRightArrow)
		}
		ib.Print()
		if i == len(ls)-1 {
			// Last item, print a final arrow
			// 49 = default BG colour
			printBashColor(int(ib.Background-10), DefaultBackgroundColour, false)
			fmt.Print(CharRightArrow)
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

func makeUserBlock(usr *user.User) *InfoBlock {
	block := makeBlock(color.FgHiYellow, color.BgBlack, usr.Username)

	uid, _ := strconv.Atoi(usr.Uid)

	if uid == 0 {
		// we are root
		block.Text = CharLightning + " " + block.Text
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
			gitBlock.Text = CharGitDetached + " " + head
		} else {
			gitBlock.Text = CharGitBranch + " " + head
		}
	}

	// Change output if repo is dirty
	if isDirty {
		gitBlock.Text += CharPlusMinus
		gitBlock.Background = color.BgHiYellow
	}

	// Check if we've diverged at all from remote
	if diff, ok := gitHeaders["branch.ab"]; ok {
		// we're missing some commits
		diffp := strings.Split(diff, " ")

		diffMessage := ""

		if diffp[0] != "+0" {
			// more than 0 extra local commits
			diffMessage = CharUpArrow + diffp[0][1:]
		}
		if diffp[1] != "-0" {
			// more than 0 extra remote commits
			if len(diffMessage) > 0 {
				diffMessage += " "
			}
			diffMessage += CharDownArrow + diffp[1][1:]
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

	useTwoLineLayout := os.Getenv("PS1_TWO_LINE") != ""

	// Check for the last bash return code
	// We pass this variable to this program from Bash as there
	// is no convenient way to pass it otherwise
	var returnCodeBlock *InfoBlock
	if len(os.Args) > 1 {
		code := os.Args[1]
		if code != "0" {
			returnCodeBlock = makeBlock(color.FgHiBlack, color.BgHiRed, CharFailure+" "+code)
		} else {
			returnCodeBlock = makeBlock(color.FgHiBlack, color.BgHiGreen, CharSuccess)
		}
	}

	if !useTwoLineLayout && returnCodeBlock != nil {
		list = append(list, returnCodeBlock)
	}

	// Add the block for a non-default user
	currentUser, _ := user.Current()
	list = append(list, makeUserBlock(currentUser))

	// Path to CWD block
	cwdB := makeBlock(color.FgHiBlack, color.BgHiBlue, getCwd())
	list = append(list, cwdB)

	// Git block
	gitB := gitBlock()
	if gitB != nil {
		list = append(list, gitB)
	}

	// Print the list
	fmt.Printf("\n\001\033[2K\002") // go to a new line and clear it for us
	printBlockList(list)

	if useTwoLineLayout {
		if returnCodeBlock.Background == color.BgHiGreen && currentUser.Uid == "0" {
			// We are root and successful, change bg to yellow
			returnCodeBlock.Background = color.BgHiYellow
		}

		fmt.Print("\n")
		returnCodeBlock.Print()
		printBashColor(int(returnCodeBlock.Background-10), DefaultBackgroundColour, false)
		fmt.Print(CharRightArrow)
	}

	// Reset the colour and print the 'K' ANSI control code
	// which is "Erase in Line".
	// K = clear from cursor to end
	// 1K = clear from cursor to start of line
	// 2K = clear entire line
	//spaceC := color.New(color.Reset)
	fmt.Print("\001\033[0m\002 \001\033[K\002")

	//color.Unset() // Reset back for the command input
}
