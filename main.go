package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf16"
)

func guessPassword(filePath string) string {
	isA, _ := regexp.MatchString(`A\d{3}`, filePath)
	isB, _ := regexp.MatchString(`B\d{3}`, filePath)
	isN, _ := regexp.MatchString(`N\d{3}`, filePath)

	if isA {
		return "忧郁的弟弟"
	} else if isB {
		return "忧郁的loli"
	} else if isN {
		return "终点"
	} else {
		fmt.Println("Auto extract/repair not supported for those series...")
		fmt.Scanln()
		os.Exit(0)
		return ""
	}
}

func Exec(cmd *exec.Cmd) {
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	cmd.Start()

	go func() {
		in := bufio.NewScanner(stderr)
		for in.Scan() {
			fmt.Println(in.Text())
		}
	}()

	in := bufio.NewScanner(stdout)
	for in.Scan() {
		fmt.Println(in.Text())
	}

	cmd.Wait()
}

func testRar(fPath string) {
	errPath := filepath.Clean(path.Join(filepath.Dir(fPath), "error.txt"))
	cmd := exec.Command("rar", "-ilog"+errPath, "t", "-p"+guessPassword(fPath), fPath)
	Exec(cmd)
}

func recoverRar(fPath string) {
	cmd := exec.Command("rar", "r", "-p"+guessPassword(fPath), fPath)
	Exec(cmd)
}

func extractRar(fDir, fPath string) {
	cmd := exec.Command("rar", "x", "-p"+guessPassword(fPath), fPath, fDir)
	Exec(cmd)
}

func getRarLog(errPath string) []string {
	b, err := os.ReadFile(errPath)
	if err != nil {
		fmt.Println(err)
		fmt.Scanln()
		os.Exit(0)
	}

	txt, err := DecodeUTF16(b)
	if err != nil {
		fmt.Println(err)
		fmt.Scanln()
		os.Exit(0)
	}

	lines := strings.Split(txt, "\r\n")

	return lines
}

func checkRarIsNeedRecover(fPath string, lines []string) []string {
	needRecover := []string{}

	for _, line := range lines {
		base := filepath.Base(line)
		ext := filepath.Ext(line)
		recPath := filepath.Clean(path.Join(fPath, base))
		if ext == ".rar" {
			needRecover = append(needRecover, recPath)
			recoverRar(recPath)
			if _, err := os.Stat("fixed." + base); errors.Is(err, os.ErrNotExist) {
				fmt.Println(base, "is damaged and unrepairable.")
				fmt.Scanln()
				os.Exit(0)
			}
			os.Rename(recPath, recPath+".backup")
			os.Rename("fixed."+base, recPath)
		}
	}

	return needRecover
}

func DecodeUTF16(b []byte) (string, error) {
	ints := make([]uint16, len(b)/2)
	if err := binary.Read(bytes.NewReader(b), binary.LittleEndian, &ints); err != nil {
		return "", err
	}
	return string(utf16.Decode(ints)), nil
}

func main() {
	if len(os.Args) == 1 {
		fmt.Println("Please drag and drop rar file to executable!")
		fmt.Scanln()
		return
	}

	curPath := os.Getenv("PATH")
	err := os.Setenv("PATH", curPath+";C:\\Program Files\\WinRAR\\;C:\\Program Files (x86)\\WinRAR\\")
	if err != nil {
		fmt.Println(err)
		fmt.Scanln()
		return
	}

	file := os.Args[1]
	fDir := filepath.Dir(file)
	fName := filepath.Base(file)
	fExt := filepath.Ext(file)
	errPath := filepath.Clean(path.Join(fDir, "error.txt"))

	os.Remove(errPath)

	if fExt != ".rar" {
		fmt.Printf("rar file only (Current: %v)", fExt)
		fmt.Scanln()
		return
	}

	testRar(file)

	if _, err := os.Stat(errPath); errors.Is(err, os.ErrNotExist) {
		extractRar(fDir, file)
		fmt.Printf("File is fine.")
		fmt.Scanln()
		return
	}

	fmt.Println("Some file is damaged, trying to repair...")
	lines := getRarLog(errPath)

	needRecover := checkRarIsNeedRecover(fDir, lines)
	if len(needRecover) > 0 {
		extractRar(fDir, needRecover[0])
	}

	fmt.Println(fName, "is repaired and extracted to", fDir)
	fmt.Scanln()
}
