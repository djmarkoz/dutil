// Copyright © 2018 Mark Freriks <m.freriks@avisi.nl>
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var sourceRepository string
var sourceVersion string
var destRepository string
var destVersion string
var imageNameFilter string
var imageNameExcludeFilter string

// retagCmd represents the retag command
var retagCmd = &cobra.Command{
	Use:   "retag",
	Short: "Change the repository of existing Docker images",
	Run: func(cmd *cobra.Command, args []string) {
		retag()
	},
}

func init() {
	rootCmd.AddCommand(retagCmd)
	retagCmd.PersistentFlags().StringVarP(&imageNameFilter, "filter", "f", "", "filter Docker images by name; name must contain the provided filter")
	retagCmd.PersistentFlags().StringVarP(&imageNameExcludeFilter, "exclude-filter", "e", "", "exclude filter Docker images by name; name must not contain the provided exclude filter")

	retagCmd.PersistentFlags().StringVarP(&sourceRepository, "source-repository", "s", "", "source Docker image repository")
	retagCmd.PersistentFlags().StringVarP(&sourceVersion, "source-version", "v", "latest", "source Docker image version")
	retagCmd.PersistentFlags().StringVarP(&destRepository, "dest-repository", "d", "", "destination Docker repository")
	retagCmd.PersistentFlags().StringVarP(&destVersion, "dest-version", "V", "latest", "destination Docker image version")

	viper.BindPFlag("source-repository", retagCmd.PersistentFlags().Lookup("source-repository"))
	viper.BindPFlag("source-version", retagCmd.PersistentFlags().Lookup("source-version"))
	viper.BindPFlag("dest-repository", retagCmd.PersistentFlags().Lookup("dest-repository"))
	viper.BindPFlag("dest-version", retagCmd.PersistentFlags().Lookup("dest-version"))
}

func retag() {
	cmd := exec.Command("docker", "image", "ls", "--format", "{{.ID}} {{.Repository}} {{.Tag}}")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal("failed to list docker images", err)
	}
	if err := cmd.Start(); err != nil {
		log.Fatal("failed to start listing docker images", err)
	}

	workChannel := make(chan string, 256)
	var wg sync.WaitGroup
	for i := 0; i < threads; i++ {
		go func() {
			for imageToPush := range workChannel {
				pushImage(imageToPush)
				wg.Done()
			}
		}()
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		processLine(&wg, workChannel, line)
	}
	if err := scanner.Err(); err != nil {
		log.Fatal("reading standard input:", err)
	}
	if err := cmd.Wait(); err != nil {
		log.Fatal("failed to wait for listing docker images", err)
	}

	wg.Wait()
	close(workChannel)
}

func processLine(wg *sync.WaitGroup, workChannel chan<- string, line string) {
	if strings.Contains(line, " "+sourceVersion) && strings.Contains(line, sourceRepository) {

		parts := strings.Split(line, " ")
		if len(parts) != 3 {
			log.Fatal(errors.New("Incorrect output of 'docker images ls'"))
		}
		url := strings.Replace(parts[1], sourceRepository, destRepository, 1)
		version := strings.Replace(parts[2], sourceVersion, destVersion, 1)

		if imageNameFilter != "" && !strings.Contains(url, imageNameFilter) {
			return
		}
		if imageNameExcludeFilter != "" && strings.Contains(url, imageNameExcludeFilter) {
			log.Println("exclude image: '" + line + "'")
			return
		}

		image := fmt.Sprintf("%s:%s", url, version)
		log.Println(image)
		cmd := exec.Command("docker", "image", "tag", parts[0], image)
		err := cmd.Start()
		if err != nil {
			log.Fatal("failed to tag docker image: ", err)
		}
		err = cmd.Wait()
		if err != nil {
			log.Fatal("failed to wait for docker tag command: ", err)
		}

		wg.Add(1)
		workChannel <- image
	}
}

func pushImage(image string) {
	log.Printf("Pushing Docker image %s\n", image)

	cmd := exec.Command("docker", "push", image)
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("%s\n", stdoutStderr)
		log.Fatal("failed", err)
	}
	log.Printf("Pushed %s to GCR\n", image)
}
