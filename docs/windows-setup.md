# Nodevin Setup on Windows

This guide will walk you through the steps of setting up **Nodevin** on your Windows machine, ensuring you have the proper Docker and Docker Compose versions installed.

## Installation Steps

### 1. Download the Nodevin Package

First, download the latest Windows release (**nodevin-windows-amd64-\<version\>.zip**, or **nodevin-windows-arm64-\<version\>.zip** for ARM64 devices) from [here](https://github.com/fiftysixtech/nodevin/releases).

### 2. Extract the Package

Once downloaded, extract the contents of the zip file to a directory of your choice.

### 3. Open Command Prompt

To proceed, open the **Command Prompt**:
- You can search for "cmd" in the Start Menu search bar, or
- Open Command Prompt manually by navigating through the Start Menu.

### 4. Navigate to the Extracted Directory

Using the Command Prompt, navigate to the directory where you extracted the Nodevin files. You can do this with the `cd` command. For example:

```bash
cd path\to\extracted\folder
```

To verify you're in the correct directory, use the `dir` command to list the files in the current directory:

```bash
dir
```

Check that `nodevin.exe` is visible in the list of files. If you see it, you are in the right place!

### 5. Verify Nodevin Version

Run the following command to check the version of **Nodevin**:

```bash
nodevin.exe version
```

If the version output matches the version you downloaded, you're on the right track.

### 6. Initialize Nodevin

Next, run the initialization process with the following command:

```bash
nodevin.exe init
```

This command will inspect your system for the necessary Docker and Docker Compose versions. If they are missing or outdated, **Nodevin** can download the latest versions for you.

### 7. Docker Installation Process

During the initialization process, you might see the following in your Command Prompt if Docker is being installed:

- A line stating **Starting Docker installation step**.
- A series of messages showing the download process, including:

  ```bash
  Writing web request...
  Writing request stream...
  ...
  Number of bytes written: [increasing number]
  ```

This output indicates that Docker is being downloaded.

### 8. Install Docker

After Docker has been properly downloaded, check your Windows **Downloads** folder for the file named `docker-desktop-installer.exe`.

- Open `docker-desktop-installer.exe` to begin the Docker installation.
- Follow the on-screen instructions provided by the Docker installer.
  
### 9. Restart Your Computer

At the end of the Docker installation, you will be asked to restart your computer. **Make sure to restart** for Docker to properly complete its setup.

### 10. Post-Restart Docker Configuration

After restarting your computer, Docker will prompt you with multiple options for signing in and specifying other configurations for Docker. **Nodevin** assumes that you are running Docker as an administrator and that you have the ability to write to disk.

> **Note**: If you do not run Docker as an admin or cannot write to disk, **Nodevin** will not be able to save blockchain data to your computer.

### 11. Finalizing Nodevin Setup

Once Docker installation is complete, **Nodevin** should be ready to go. To ensure everything is installed correctly, run the following command:

```bash
nodevin.exe init
```

If the output ends with "It's time to start your own Bitcoin node. Run `nodevin start bitcoin` to get started.", then Docker and Nodevin are correctly installed, and you're ready to start using Nodevin!
