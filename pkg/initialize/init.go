package initialize

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/fiftysixcrypto/nodevin/internal/logger"
	"github.com/fiftysixcrypto/nodevin/internal/version"
	"github.com/spf13/cobra"
)

var skipPrompt bool

var InitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize nodevin and check system capabilities",
	Run: func(cmd *cobra.Command, args []string) {
		runInit()
	},
}

func init() {
	InitCmd.Flags().BoolVarP(&skipPrompt, "yes", "y", false, "Automatically confirm all prompts")
}

func runInit() {
	fmt.Println("")
	fmt.Printf("Welcome to nodevin v%s!\n", version.Version)
	fmt.Println("Nodevin makes running blockchain nodes more accessible.")
	fmt.Println("")
	fmt.Println("--")
	fmt.Println("")

	// inspection
	fmt.Println("Nodevin will now inspect your system for docker and docker compose versions...")
	if err := performInspection(); err != nil {
		fmt.Println("")
		logger.LogError("System inspection failed: " + err.Error())

		// Skip prompt if `-y` is set
		if skipPrompt {
			fmt.Println("Skipping prompt and proceeding with Docker and Docker Compose installation...")
			if err := installDockerAndCompose(); err != nil {
				logger.LogError("Failed to install Docker and Docker Compose: " + err.Error())
				return
			}
		} else {
			// Ask the user if they want to install Docker and Docker Compose
			reader := bufio.NewReader(os.Stdin)
			fmt.Print("Docker and Docker Compose are required but not installed. Would you like to install them? ('y'/'n'): ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))

			// Check user response
			if input == "y" {
				if err := installDockerAndCompose(); err != nil {
					logger.LogError("Failed to install Docker and Docker Compose: " + err.Error())
					return
				}
			} else {
				fmt.Println("Canceling docker installation.")
				return
			}
		}
	}

	fmt.Println("")
	fmt.Println("--")
	fmt.Println("")

	// outro
	fmt.Println("It's time to start your own Bitcoin node. Run `nodevin start bitcoin` to get started.")
	fmt.Println("Thank you for using nodevin!")
}

func performInspection() error {
	fmt.Print("Checking Docker version... ")
	if err := checkDockerVersion(); err != nil {
		return err
	}

	fmt.Print("Checking Docker Compose version... ")
	if err := checkDockerComposeVersion(); err != nil {
		return err
	}

	return nil
}

func checkDockerVersion() error {
	cmd := exec.Command("docker", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to check Docker version: %w", err)
	}

	versionInfo := string(output)
	fmt.Print(versionInfo)

	re := regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(versionInfo)
	if len(matches) < 4 {
		return fmt.Errorf("unable to parse Docker version")
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	if major < 20 {
		return fmt.Errorf("Docker version 20+ is required. Please update Docker.")
	}

	fmt.Printf("Docker version %d.%d.%d meets the requirement.\n", major, minor, patch)
	return nil
}

func checkDockerComposeVersion() error {
	cmd := exec.Command("docker-compose", "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to check Docker Compose version: %w", err)
	}

	versionInfo := string(output)
	fmt.Print(versionInfo)

	re := regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(versionInfo)
	if len(matches) < 4 {
		return fmt.Errorf("unable to parse Docker Compose version")
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	if major < 1 {
		return fmt.Errorf("Docker Compose version 1+ is required. Please update Docker Compose.")
	}

	fmt.Printf("Docker Compose version %d.%d.%d meets the requirement.\n", major, minor, patch)
	return nil
}

func installDockerAndCompose() error {
	fmt.Println("Installing Docker and Docker Compose...")

	if err := installDocker(); err != nil {
		return fmt.Errorf("failed to install Docker: %w", err)
	}

	if err := installDockerCompose(); err != nil {
		return fmt.Errorf("failed to install Docker Compose: %w", err)
	}

	switch runtime.GOOS {
	case "linux":
		fmt.Println(`
Docker installation complete! 
Make sure Docker is running with the following command:
   sudo systemctl start docker
Then, confirm the installation:
   docker --version`)

	case "darwin":
		fmt.Println(`
Download complete.
Next steps to finish Docker setup:
1. Drag "Docker Desktop" from Downloads/Docker.dmg into "Applications".
2. Open Docker Desktop and go through the options. Make sure it is running in the top right corner of your screen. 
3. Open your terminal and run the following command to verify Docker installation:
   docker --version
4. After confirming Docker is running, you can start using Nodevin. Remember, Nodevin will only work if Docker Desktop is running!`)

	case "windows":
		fmt.Println(`
Download complete.
Next steps to finish Docker setup:
1. Navigate to your downloads folder and find the file "docker-desktop-installer.exe".
2. Open the installer.
3. Follow the instructions to install Docker on your computer.
4. Click "Close and restart" to restart your computer and finish the installation.
5. After confirming Docker is running and working, you can start using Nodevin. Remember, Nodevin will only work if Docker Desktop is running!`)
	}

	fmt.Println("Docker and Docker Compose installed successfully.")
	return nil
}

func installDocker() error {
	var installCmd *exec.Cmd
	var downloadsPath string
	arch := runtime.GOARCH

	switch runtime.GOOS {
	case "windows":
		downloadsPath = filepath.Join(os.Getenv("USERPROFILE"), "Downloads")
	case "darwin", "linux":
		downloadsPath = filepath.Join(os.Getenv("HOME"), "Downloads")
	}

	switch runtime.GOOS {
	case "linux":
		fmt.Printf("Starting Docker installation for Linux (%s)...\n", arch)
		installCmd = exec.Command("sh", "-c", fmt.Sprintf(`
			sudo apt update &&
			sudo apt install -y apt-transport-https ca-certificates curl software-properties-common &&
			curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo apt-key add - &&
			sudo add-apt-repository "deb [arch=%s] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" &&
			sudo apt update &&
			sudo apt install -y docker-ce &&
   			sudo usermod -aG docker $USER &&
			newgrp docker`, arch))

	case "darwin":
		fmt.Printf("Starting Docker download for macOS (%s)...\n", arch)
		var dockerURL string
		if arch == "amd64" {
			dockerURL = "https://desktop.docker.com/mac/stable/amd64/Docker.dmg"
		} else if arch == "arm64" {
			dockerURL = "https://desktop.docker.com/mac/stable/arm64/Docker.dmg"
		} else {
			return fmt.Errorf("unsupported architecture: %s", arch)
		}

		installCmd = exec.Command("sh", "-c", fmt.Sprintf(`
			echo "Downloading Docker for macOS..." &&
			curl -L %s -o %s/Docker.dmg &&
			echo "Opening Docker installer..." &&
			open %s/Docker.dmg &&
			echo "Once Docker is installed, open Docker Desktop before running Nodevin commands."`, dockerURL, downloadsPath, downloadsPath))

	case "windows":
		fmt.Printf("Starting Docker download for Windows (%s)...\n", arch)
		var dockerURL string
		if arch == "amd64" {
			dockerURL = "https://desktop.docker.com/win/main/amd64/Docker%20Desktop%20Installer.exe"
		} else if arch == "arm64" {
			dockerURL = "https://desktop.docker.com/win/main/arm64/Docker%20Desktop%20Installer.exe"
		} else {
			return fmt.Errorf("unsupported architecture: %s", arch)
		}

		installCmd = exec.Command("powershell", "-Command", fmt.Sprintf(`
			$ErrorActionPreference = 'Stop';
			$downloadsFolder = "%s";
			Write-Host 'Downloading Docker for Windows...';
			Invoke-WebRequest -UseBasicParsing -OutFile "$downloadsFolder\\docker-desktop-installer.exe" %s -Verbose;
			Write-Host 'Once Docker is installed, open Docker Desktop before running Nodevin commands.';`, downloadsPath, dockerURL))

	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	if err := installCmd.Run(); err != nil {
		return err
	}

	return nil
}

func installDockerCompose() error {
	var installCmd *exec.Cmd

	fmt.Println("Installing Docker Compose...")

	switch runtime.GOOS {
	case "linux":
		installCmd = exec.Command("sh", "-c", `
			sudo curl -L "https://github.com/docker/compose/releases/download/v2.27.1/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose &&
			sudo chmod +x /usr/local/bin/docker-compose &&
			docker-compose --version`)
	case "darwin", "windows":
		fmt.Println("Docker Compose is already installed with Docker Desktop on macOS and Windows.")
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	if runtime.GOOS == "linux" {
		if err := installCmd.Run(); err != nil {
			return err
		}
		fmt.Println("Docker Compose installation complete for Linux.")
	}

	return nil
}
