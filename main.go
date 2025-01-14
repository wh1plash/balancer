package main

import (
	"balancer_my/internal"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	fileMutex     sync.Mutex
	fileFirstSeen = make(map[string]time.Time)
)

func main() {
	cfg := internal.MustLoad()
	fmt.Printf("Load config params:\n{\n Direcory for monitoring: %s\n Direcory for balacing: %s\n Number of folders: %d\n}\n", cfg.SrcDir, cfg.DestDir, cfg.NumFolders)

	fmt.Println("Startin balancer app")

	fileChan := make(chan string)
	go watchFiles(fileChan, cfg)

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			//defer wg.Done()
			moveFile(fileChan, &wg, cfg)
		}()
	}

	wg.Wait()

}

func moveFile(fileChan <-chan string, wg *sync.WaitGroup, cfg *internal.Config) {
	for fileInChan := range fileChan {
		defer wg.Done()
		fmt.Println("Starting to move file...", fileInChan)
		destPath := balanceFolder(cfg.DestDir, fileInChan, cfg.NumFolders)

		targetPath := filepath.Join(destPath, filepath.Base(fileInChan))

		file, err := os.Create(targetPath)
		if err != nil {
			fmt.Println("Can't create file:", file)
		}
		defer file.Close()

		srcFile, err := os.Open(fileInChan)
		if err != nil {
			fmt.Println("Can't open file:", srcFile)
		}

		if _, err := io.Copy(file, srcFile); err != nil {
			fmt.Println("Can't copy file:", srcFile)
		}
		srcFile.Close()

		err = os.Remove(fileInChan)
		if err != nil {
			fmt.Println("Can't remove file:", fileInChan)
		}

		fmt.Printf("File %s moved successfully to %s\n", srcFile.Name(), targetPath)
	}
}

func balanceFolder(destDir string, path string, numDirs int) string {
	h := fnv.New32a()
	h.Write([]byte(path))
	folderIndex := h.Sum32() % uint32(numDirs)
	targetPath := filepath.Join(destDir, fmt.Sprintf("part_%d", folderIndex))

	if err := os.MkdirAll(targetPath, os.ModePerm); err != nil {
		log.Fatal("Can't create folder:", targetPath)
	}

	return targetPath
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
