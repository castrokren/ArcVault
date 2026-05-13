package main

import (
"bufio"
"crypto/rand"
"fmt"
"os"
"strconv"
"strings"
)

func main() {
if len(os.Args) < 2 {
fmt.Println("Usage: coordinator init|start|help")
os.Exit(1)
}

switch os.Args[1] {
case "init":
doInit()
case "start":
fmt.Println("Server would start on :8080")
case "help":
fmt.Println("ArcVault Coordinator - init|start|help")
}
}

func doInit() {
fmt.Println("ArcVault Coordinator - Initialization")
fmt.Println("=====================================")
fmt.Println()

reader := bufio.NewReader(os.Stdin)

fmt.Print("Enter port (default 8080): ")
portStr, _ := reader.ReadString('\n')
portStr = strings.TrimSpace(portStr)
port := 8080
if p, err := strconv.Atoi(portStr); err == nil {
port = p
}

fmt.Print("Enter database path (default ~/.arcvault/arcvault.db): ")
dbPath, _ := reader.ReadString('\n')
dbPath = strings.TrimSpace(dbPath)
if dbPath == "" {
homeDir, _ := os.UserHomeDir()
dbPath = homeDir + "\\.arcvault\\arcvault.db"
}

token := generateToken(32)

fmt.Printf("\nPort: %d\n", port)
fmt.Printf("Database: %s\n", dbPath)
fmt.Printf("Admin token (save this): %s\n\n", token)
fmt.Println("Next step: Run 'coordinator start'")
}

func generateToken(length int) string {
bytes := make([]byte, length)
rand.Read(bytes)
hexStr := ""
for _, b := range bytes {
hexStr += fmt.Sprintf("%02x", b)
}
return hexStr
}