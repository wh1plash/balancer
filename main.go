package main

import (
	"balancer_my/internal"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"log"
)

var (
	fileMutex     sync.Mutex
	fileFirstSeen = make(map[string]time.Time)
)

func main() {
	cfg := internal.MustLoad()
	fmt.Printf("Load config params:\n{\n Direcory for monitoring: %s\n Direcoryes for balacing: %s\n}\n", cfg.SrcDir, cfg.Folders)

	fmt.Println("Starting balancer app")

	fileChan := make(chan string)
	go watchFiles(fileChan, cfg)

	var wg sync.WaitGroup
	for i := 0; i < len(cfg.Folders); i++ {
		wg.Add(1)
		go func(i int) {
			moveFile(fileChan, &wg, cfg.Folders[i])
		}(i)
	}

	wg.Wait()

}

func moveFile(fileChan <-chan string, wg *sync.WaitGroup, folder string) {
	if err := os.MkdirAll(folder, os.ModePerm); err != nil {
		log.Fatalf("failed to create data directory: %s", folder)
	}

	for fileInChan := range fileChan {
		defer wg.Done()
		fmt.Println("Starting to move file...", fileInChan)

		targetPath := filepath.Join(folder, filepath.Base(fileInChan))

		file, err := os.Create(targetPath)
		if err != nil {
			fmt.Println("Can't create file:", file)
		}
		

		srcFile, err := os.Open(fileInChan)
		if err != nil {
			fmt.Println("Can't open file:", srcFile)
		}

		if _, err := io.Copy(file, srcFile); err != nil {
			fmt.Println("Can't copy file:", srcFile)
		}
		srcFile.Close()
		file.Close()
		
		err = os.Remove(fileInChan)
		if err != nil {
			fmt.Println("Can't remove file:", fileInChan)
		}

		fmt.Printf("File %s moved successfully to %s\n", srcFile.Name(), targetPath)
	}
}

func watchFiles(fileChan chan<- string, cfg *internal.Config) {
	fmt.Printf("Start monitoring folder: %s\n", cfg.SrcDir)
	for {
		files, err := os.ReadDir(cfg.SrcDir)
		if err != nil {
			fmt.Printf("error reading the directory: %s\n", err)
			time.Sleep(1 * time.Second)
			continue
		}

		currentFiles := make(map[string]bool)

		for _, file := range files {
			if !file.IsDir() {
				filePath := filepath.Join(cfg.SrcDir, file.Name())
				currentFiles[filePath] = true

				fileMutex.Lock()
				if _, exists := fileFirstSeen[filePath]; !exists {
					fileFirstSeen[filePath] = time.Now()
					fmt.Printf("New file detected: %s\n", filePath)
				}
				fileMutex.Unlock()

				if isFileUnchanged(filePath) {
					fmt.Printf("File %s has not been modified for more than 10 seconds. Moving...\n", filePath)
					fileChan <- filePath // Отправляем файл в канал
				} else {
					fmt.Printf("File %s is not ready to be moved yet \n", filePath)
				}
			}
		}

		// Удаляем из карты файлы, которых больше нет в директории
		fileMutex.Lock()
		for filePath := range fileFirstSeen {
			if !currentFiles[filePath] {
				delete(fileFirstSeen, filePath)
				fmt.Printf("File has been removed from tracking: %s\n", filePath)
			}
		}
		fileMutex.Unlock()
		time.Sleep(1 * time.Second)

	}
}

func isFileUnchanged(filePath string) bool {
	fileMutex.Lock()
	firstSeen, exists := fileFirstSeen[filePath]
	fileMutex.Unlock()

	if !exists {
		return false
	}

	return time.Since(firstSeen) > 10*time.Second
}
