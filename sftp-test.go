package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"sync"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"path/filepath"
	"io/ioutil"
	"io"
)

var confDir = flag.String("remote-dir", "/", "Remote directory to search")
var confHost = flag.String("h", "", "SFTP server hostname or IP address")
var confPort = flag.String("p", "22", "SFTP server port")
var confUser = flag.String("U", "Admin", "Username")
var confPwd = flag.String("P", "Admin", "Password")
var confRegex = flag.String("r", "", "Regex for file search")
var confRename = flag.Bool("rename", false, "Enable renaming matched files")
var confInterval = flag.Uint("I", 1, "Interval length between requests [seconds]")
var confTotal = flag.Uint("T", 1, "Total requests to send")
var confDst = flag.String("local-dir", "", "Local directory to save files (use with -read)")
var confGetFile = flag.Bool("read", false, "Read and save the remote file(s)")
var confRenameSuffix = flag.String("rename-suffix", "", "Rename suffix (use with -rename)")
var confVersion = flag.Bool("version", false, "Print version")

const VERSION_STR = "SFTP-TEST version 1.2"

func main() {
	flag.Parse()
	
	if *confVersion {
		fmt.Println(VERSION_STR)
		return
	}

	var regex *regexp.Regexp
	if len(*confRegex) > 0 {
		regex = regexp.MustCompile(*confRegex)
	}

	if !validateConfiguration() {
		os.Exit(1)
	}

	// Configure the SSH connection
	config := &ssh.ClientConfig{
		User: *confUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(*confPwd),
		},
	}

	// Create an SSH connection
	fmt.Printf("Opening SSH connection to %s on port %s...", *confHost, *confPort)
	conn, err := ssh.Dial("tcp", *confHost + ":" + *confPort, config)
	if err != nil {
		fmt.Println(" FAIL")
		log.Fatal(err)
	}
	defer conn.Close()
	fmt.Println(" OK")

	// open an SFTP session over the existing SSH connection
	fmt.Printf("Opening SFTP session...")
	sftpc, err := sftp.NewClient(conn)
	if err != nil {
		fmt.Println(" FAIL")
		log.Fatal(err)
	}
	defer sftpc.Close()
	fmt.Println(" OK")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var total = *confTotal
		interval := time.Duration(*confInterval) * time.Second

		for total > 0 {
			// walk a directory
			fmt.Println("Reading directory: ", *confDir)
			files, err := readDirectory(sftpc, *confDir, regex)
			if err != nil {
				log.Printf("Error: %v\n", err)
				break
			}

			// print matched files
			fmt.Println("Matched files: ", len(files))
			for _, f := range files {
				fmt.Println("\t", f)
			}

			// read (copy) files
			if *confGetFile {
				for _, f := range files {
					fmt.Printf("reading file: %s...\n", f)
					getFile(sftpc, f, *confDir)
				}
			}

			// rename files
			if *confRename {
				for _, f := range files {
					oldname := *confDir + f
					newname := *confDir + f+*confRenameSuffix
					fmt.Println("Renaming ", oldname)
					err := sftpc.Rename(oldname, newname)
					if err != nil {
						log.Printf("Rename error: (%s --> %s) %v\n", oldname, newname, err)
					}
				}
			}

			total--
			if total == 0 {
				break
			}
			log.Printf("Iterations left: %d\n", total)
			time.Sleep(interval)
		}
	}()

	wg.Wait()
}

func getFile(sftpc *sftp.Client, f string, dir string) {
	srcFile, err := sftpc.Open(dir + f)
	if err != nil {
		log.Printf("Error opening file %s (%v)", f, err)
		return
	}
	defer srcFile.Close()

	dst := filepath.Join(*confDst, srcFile.Name())
	err = os.MkdirAll(filepath.Dir(dst), 777)
	if err != nil {
		log.Printf("Failed to create %s: %v\n", filepath.Dir(dst), err)
		return
	}

	b := make([]byte, 10*1024*1024)
	n, err := srcFile.Read(b)
	if err != nil && err != io.EOF {
		log.Printf("Error reading file %s (%v)\n", f, err)
		return
	}

	err = ioutil.WriteFile(dst, b[:n], 0600)
	if err != nil {
		log.Printf("Error writing file %s (%v)\n", dst, err)
		return
	}
	fmt.Printf("Written file: %s (%d bytes)", dst, n)
}

func readDirectory(client *sftp.Client, dir string, regex *regexp.Regexp) ([]string, error) {
	var files []string

	finfo, err := client.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, info := range finfo {
		if info.IsDir() {
			continue
		}
		if regex != nil {
			match := regex.MatchString(info.Name())
			if match {
				files = append(files, info.Name())
			}
		} else {
			files = append(files, info.Name())
		}
	}

	return files, nil
}

func validateConfiguration() bool {

	if len(*confHost) == 0 {
		log.Println("Missing host")
		return false
	}
	if !govalidator.IsPort(*confPort) {
		log.Printf("Illegal port: %s", *confPort)
		return false
	}

	var err error
	if len(*confDst) == 0 {
		*confDst, err = os.Getwd()
		if err != nil {
			log.Printf("Failed to get current working directory: %v", err)
			*confDir = os.TempDir()
		}
	}

	if *confRename && len(*confRenameSuffix) == 0 {
		log.Printf("Rename suffix is mandatory when using -rename option")
		return false
	}

	return true
}