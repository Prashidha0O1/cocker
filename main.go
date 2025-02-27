//go:build linux

package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Define global variables that we'll use throughout our program
// In real-world applications, it might be better to use configuration files
// but for simplicity and learning purposes, we'll use globals here
var containersDirectory string = "./containers"  // This is where we'll store all our container filesystems
var rootFileSystemTarball string = "./ubuntu-base-22.04-base-amd64.tar.gz"  // This is the base image we'll use for our containers

// The init() function runs before main() and is used to set up any prerequisites
// This is like the constructor in object-oriented programming languages!
func init() {
	// We need to create the containers directory if it doesn't exist already
	// This is where we'll store all of our container data
	// The 0700 permission means only the owner can read, write, and execute
	err := os.MkdirAll(containersDirectory, 0700)
	
	// If there's an error creating the directory, we'll quit the program
	// This is a critical error since we can't proceed without a place to store containers
	if err != nil {
		// log.Fatal will print the error and then exit the program with a non-zero status code
		log.Fatal("Failed to create containers directory during initialization: ", err)
	}
	
	// Let's print a message to confirm that initialization was successful
	// This isn't strictly necessary but it's helpful for debugging and learning
	fmt.Println("===== Container Runtime Initialized =====")
	fmt.Println("Container storage location:", containersDirectory)
	fmt.Println("Using root filesystem:", rootFileSystemTarball)
	fmt.Println("=========================================")
}

// The main function is the entry point of our program
// Every Go program needs exactly one main() function in the main package
func main() {
	// First, we need to check if the user provided a command
	// os.Args[0] is the program name itself, so we need at least one more argument
	if len(os.Args) < 2 {
		// If we don't have enough arguments, print an error message and exit
		log.Fatal("Error: You must provide a command! Try 'run' to create a container or 'ps' to list containers.")
	}

	// Let's get the command from the arguments
	// The command will be the first argument after the program name
	userCommand := os.Args[1]
	
	// Now we'll use a switch statement to handle different commands
	// A switch statement is like a series of if-else statements but more readable
	switch userCommand {
	case "run":
		// The "run" command creates a new container
		// We pass false to indicate this is the parent process (not inside the container)
		fmt.Println("Starting container creation process...")
		handleContainerCreation(false)
		
	case "_child":
		// This is a special internal command that's used when we're inside the container
		// We pass true to indicate this is the child process (inside the container)
		fmt.Println("Initializing container environment...")
		handleContainerCreation(true)
		
	case "ps":
		// The "ps" command lists all existing containers
		fmt.Println("Listing all containers...")
		listAllContainers()
		
	default:
		// If the command isn't recognized, show an error
		log.Fatal("Error: Unknown command '" + userCommand + "'. Valid commands are 'run' (to create a container) and 'ps' (to list containers).")
	}
}

// This function handles the container creation process for both the parent and child processes
// We split the logic into a separate function to make the code more organized and easier to understand
func handleContainerCreation(isChildProcess bool) {
	// We need to parse the command-line arguments to extract volume mappings and the command to run
	// Volumes let us share directories between the host and the container
	var volumeMappings []string  // This will store our volume mappings (host:container format)
	var commandArguments []string  // This will store the command and its arguments
	
	// Let's print what arguments we're working with
	fmt.Println("Parsing command arguments:", os.Args[2:])

	// We start from index 2 because os.Args[0] is the program name and os.Args[1] is the "run" command
	for argumentIndex := 2; argumentIndex < len(os.Args); argumentIndex++ {
		currentArgument := os.Args[argumentIndex]
		
		// Check if this argument is a volume mapping (it starts with "-v=")
		if strings.HasPrefix(currentArgument, "-v=") {
			// Extract the volume mapping part (after the "-v=")
			volumeMapping := strings.TrimPrefix(currentArgument, "-v=")
			
			// Add this mapping to our list of volume mappings
			volumeMappings = append(volumeMappings, volumeMapping)
			
			// Let's print the volume mapping we found
			fmt.Println("Detected volume mapping:", volumeMapping)
		} else {
			// If it's not a volume mapping, it's part of the command to run in the container
			commandArguments = append(commandArguments, currentArgument)
		}
	}
	
	// Print a summary of what we've parsed
	fmt.Println("Volume mappings:", volumeMappings)
	fmt.Println("Command arguments:", commandArguments)
	
	// Now let's call our function to actually run the container
	startContainer(commandArguments, volumeMappings, isChildProcess)
}

// This function actually starts the container with the specified command and volume mappings
// The isChildProcess parameter tells us whether we're running inside the container or not
func startContainer(commandArgs []string, volumeMappings []string, isChildProcess bool) {
	// First, let's check if we have any command to run
	// We need at least one argument (the command to run in the container)
	if len(commandArgs) == 0 {
		log.Fatal("Error: You must specify a command to run in the container! Example: ./container run /bin/bash")
	}

	// We'll use these variables to store the command and its arguments
	var executableToRun string  // The command to execute
	var executableArguments []string  // The arguments for that command
	
	// The logic is different depending on whether we're inside the container or not
	if isChildProcess {
		// If we're inside the container (child process), we'll run the user's command directly
		fmt.Println("Child process: preparing to execute user command inside container")
		
		// The first argument is the command to run
		executableToRun = commandArgs[0]
		
		// The rest are arguments to that command
		if len(commandArgs) > 1 {
			executableArguments = commandArgs[1:]
		} else {
			// If there are no additional arguments, initialize as empty slice
			executableArguments = []string{}
		}
		
		fmt.Println("Child: Will execute:", executableToRun, "with arguments:", executableArguments)
	} else {
		// If we're outside the container (parent process), we need to start a new process
		// that will become the container environment
		fmt.Println("Parent process: preparing to create container environment")
		
		// Get the path to our own executable
		ourExecutablePath, err := os.Executable()
		if err != nil {
			log.Fatal("Failed to get our own executable path: ", err)
		}
		
		// We'll run ourselves again, but with the special "_child" command
		// This is a common pattern in container runtimes - the parent process
		// sets up isolation, then the child process runs inside that isolation
		executableToRun = ourExecutablePath
		
		// Start with the special "_child" command
		executableArguments = []string{"_child"}
		
		// Add all the volume mappings back as arguments
		// We need to pass these to the child process
		for _, volumeMapping := range volumeMappings {
			executableArguments = append(executableArguments, "-v="+volumeMapping)
		}
		
		// Add the user's command and arguments
		executableArguments = append(executableArguments, commandArgs...)
		
		fmt.Println("Parent: Will execute:", executableToRun, "with arguments:", executableArguments)
	}

	// Now we'll create the command to execute
	// This is using Go's os/exec package to manage process execution
	commandToExecute := exec.Command(executableToRun, executableArguments...)

	// We want to connect the child process's standard input, output, and error
	// to the parent process, so the user can interact with the container
	commandToExecute.Stdin = os.Stdin
	commandToExecute.Stdout = os.Stdout
	commandToExecute.Stderr = os.Stderr

	// If we're inside the container, we need to set up the container environment
	if isChildProcess {
		// First, let's generate a unique ID for this container
		// We'll use a prefix and a random string to make it more readable
		containerId := "container-" + generateRandomIdentifier(24)
		fmt.Println("\n===== CONTAINER SETUP START =====")
		fmt.Println("Container ID:", containerId)

		// Set the hostname of the container to the container ID
		// This is one of the first signs of container isolation - each container has its own hostname
		fmt.Println("Setting container hostname...")
		err := syscall.Sethostname([]byte(containerId))
		if err != nil {
			log.Fatal("Failed to set container hostname: ", err)
		}
		fmt.Println("Hostname set successfully to:", containerId)

		// Now we need to create a filesystem for this container
		// Each container gets its own copy of the root filesystem
		containerRootFsPath := filepath.Join(containersDirectory, containerId)
		fmt.Println("Creating container filesystem at:", containerRootFsPath)
		
		// Extract the root filesystem from the tarball
		fmt.Println("Extracting root filesystem from tarball...")
		extractRootFilesystem(containerRootFsPath, rootFileSystemTarball)
		fmt.Println("Root filesystem extracted successfully")

		// Now let's handle volume mounts
		// Volumes allow sharing directories between the host and container
		fmt.Println("\nSetting up volume mounts...")
		mountedVolumePaths := []string{}  // We'll use this to track which volumes we've mounted
		
		for volumeIndex, volumeMapping := range volumeMappings {
			fmt.Printf("Processing volume %d: %s\n", volumeIndex+1, volumeMapping)
			
			// Split the volume mapping into host and container paths
			// The format is host:container
			volumeParts := strings.Split(volumeMapping, ":")
			if len(volumeParts) != 2 {
				log.Fatalf("Invalid volume mapping format: %s (should be host:container)", volumeMapping)
			}

			hostPath := volumeParts[0]  // The directory on the host
			containerPath := volumeParts[1]  // Where it should appear in the container
			
			// We need to prepare the full path in the container's filesystem
			fullContainerPath := filepath.Join(containerRootFsPath, containerPath)
			
			// Create the target directory if it doesn't exist
			fmt.Println("Creating mount point directory:", fullContainerPath)
			err := os.MkdirAll(fullContainerPath, 0700)
			if err != nil {
				log.Fatal("Failed to create mount point directory: ", err)
			}

			// Now we'll mount the host directory into the container
			// This uses the MS_BIND flag to create a bind mount
			fmt.Printf("Mounting: %s -> %s\n", hostPath, fullContainerPath)
			err = syscall.Mount(hostPath, fullContainerPath, "", syscall.MS_BIND|syscall.MS_REC, "")
			if err != nil {
				log.Fatal("Failed to mount volume: ", err)
			}
			fmt.Println("Volume mounted successfully")

			// Keep track of this mounted volume for later cleanup
			mountedVolumePaths = append(mountedVolumePaths, containerPath)
		}

		// Set up a deferred function to unmount all volumes when we're done
		// Defer statements are executed in LIFO (last in, first out) order
		// when the surrounding function returns
		defer func() {
			fmt.Println("\n===== CONTAINER CLEANUP =====")
			fmt.Println("Unmounting volumes...")
			
			for _, containerPath := range mountedVolumePaths {
				// We need to get the path relative to the new root
				// After changing root, all paths are relative to the new root
				fullPath := filepath.Join("/", containerPath)
				
				fmt.Println("Unmounting:", fullPath)
				if err := syscall.Unmount(fullPath, 0); err != nil {
					// This is not fatal, just a warning
					fmt.Printf("Warning: Failed to unmount %s: %v\n", fullPath, err)
				} else {
					fmt.Println("Successfully unmounted:", fullPath)
				}
			}
			fmt.Println("Volume cleanup completed")
		}()

		// Now we need to change the root directory for this process
		// This is a key part of container isolation - the container can only see its own filesystem
		fmt.Println("\nChanging root filesystem...")
		changeContainerRoot(containerRootFsPath)
		fmt.Println("Root filesystem changed successfully")

		// We also need to mount the proc filesystem
		// This is necessary for many Linux commands to work correctly
		fmt.Println("\nMounting proc filesystem...")
		err = syscall.Mount("proc", "/proc", "proc", 0, "")
		if err != nil {
			log.Fatal("Failed to mount proc filesystem: ", err)
		}
		fmt.Println("Proc filesystem mounted successfully")
		
		// Clean up the proc filesystem when we're done
		defer func() {
			fmt.Println("Unmounting proc filesystem...")
			if err := syscall.Unmount("/proc", 0); err != nil {
				fmt.Println("Warning: Failed to unmount proc filesystem:", err)
			} else {
				fmt.Println("Proc filesystem unmounted successfully")
			}
		}()

		fmt.Println("\n===== CONTAINER READY =====")
		fmt.Printf("Container ID: %s\n", containerId)
		fmt.Printf("Container PID: %d\n", os.Getpid())
		fmt.Printf("Executing command: %s\n", executableToRun)
		fmt.Println("=========================")
	} else {
		// If this is the parent process, we need to set up namespace isolation
		// This is what makes a container isolated from the rest of the system
		fmt.Println("\nSetting up namespace isolation for container...")
		
		// We'll use the SysProcAttr struct to configure how the child process is created
		commandToExecute.SysProcAttr = &syscall.SysProcAttr{
			// Cloneflags determine what aspects of the system are isolated
			Cloneflags: 
				// UTS namespace isolates hostname and domain name
				syscall.CLONE_NEWUTS |
				// PID namespace isolates process IDs (each container has its own PID 1)
				syscall.CLONE_NEWPID |
				// Mount namespace isolates mount points
				syscall.CLONE_NEWNS,

			// Make sure mount points aren't shared with the host
			// This is important for security - changes to mounts in the container
			// shouldn't affect the host system
			Unshareflags: syscall.CLONE_NEWNS,
		}
		
		fmt.Println("Namespace isolation configured successfully")
	}

	// Now let's run the command
	fmt.Println("\nExecuting command:", executableToRun, "with arguments:", executableArguments)
	executionError := commandToExecute.Run()
	
	// Check if there was an error running the command
	if executionError != nil {
		fmt.Fprintln(os.Stderr, "Command execution failed:", executionError)
	}

	// Get the exit code from the command
	exitCode := commandToExecute.ProcessState.ExitCode()
	fmt.Println("\nCommand execution completed with exit code:", exitCode)
	
	// Exit with the same exit code as the command
	// This passes the exit code up to the parent process
	os.Exit(exitCode)
}

// This function lists all containers in the containers directory
func listAllContainers() {
	fmt.Println("\n===== CONTAINER LISTING =====")
	
	// Read the contents of the containers directory
	containerEntries, err := os.ReadDir(containersDirectory)
	if err != nil {
		log.Fatal("Failed to read containers directory: ", err)
	}

	// Check if we have any containers
	if len(containerEntries) == 0 {
		fmt.Println("No containers found.")
		fmt.Println("============================")
		return
	}

	// Print a header for the container list
	fmt.Println("CONTAINER ID\t\t\tCREATION TIME")
	fmt.Println("--------------------------------------------")
	
	// Iterate through each entry in the containers directory
	for _, containerEntry := range containerEntries {
		// Get detailed file information
		containerInfo, err := containerEntry.Info()
		if err != nil {
			// If we can't get info for this container, print a warning and continue
			fmt.Printf("Warning: Could not get info for container '%s': %v\n", containerEntry.Name(), err)
			continue
		}

		// Print the container ID and creation time
		fmt.Printf("%s\t%s\n", 
			containerEntry.Name(), 
			containerInfo.ModTime().Format(time.UnixDate))
	}
	
	fmt.Println("============================")
}

// This function generates a random string to use as part of the container ID
// We want container IDs to be unique so we don't have conflicts
func generateRandomIdentifier(length int) string {
	// If an invalid length is provided, return an empty string
	if length < 1 {
		fmt.Println("Warning: Invalid length for random identifier, returning empty string")
		return ""
	}

	// These are the characters we'll use in our random string
	// We're using letters and numbers for readability, but we could use any characters
	const characterSet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	
	// Create a slice to hold our random string
	// A slice is a dynamically-sized, flexible view into an array
	resultCharacters := make([]byte, length)
	
	// Fill the slice with random characters
	// This isn't the most efficient way to generate a random string,
	// but it's easy to understand for learning purposes
	for i := 0; i < length; i++ {
		// Pick a random character from our character set
		randomIndex := rand.Intn(len(characterSet))
		resultCharacters[i] = characterSet[randomIndex]
	}
	
	// Convert the byte slice to a string and return it
	return string(resultCharacters)
}

// This function extracts the root filesystem for a container from a tarball
func extractRootFilesystem(destinationPath string, tarballPath string) {
	// First, we need to create the destination directory
	fmt.Println("Creating root filesystem directory:", destinationPath)
	err := os.MkdirAll(destinationPath, 0700)
	if err != nil {
		log.Fatal("Failed to create root filesystem directory: ", err)
	}

	// Now we'll use the tar command to extract the tarball
	// We could implement this in Go, but using the system tar command is simpler
	fmt.Printf("Extracting tarball %s to %s...\n", tarballPath, destinationPath)
	
	// Create a command to extract the tarball
	// The -x flag extracts files, -z handles gzip compression, -f specifies the file,
	// and -C changes to the specified directory before extraction
	tarCommand := exec.Command("tar", "-xzf", tarballPath, "-C", destinationPath)
	
	// Run the command and check for errors
	extractionError := tarCommand.Run()
	if extractionError != nil {
		log.Fatal("Failed to extract root filesystem tarball: ", extractionError)
	}
	
	fmt.Println("Root filesystem extracted successfully")
}

// This function changes the root directory for the process
// This is a critical part of container isolation - it makes the container
// see only its own filesystem, not the host filesystem
func changeContainerRoot(newRootPath string) {
	// Before we can use pivot_root, the new root directory must be a mount point
	// So we'll first bind mount the new root to itself
	fmt.Println("Binding new root to itself...")
	err := syscall.Mount(newRootPath, newRootPath, "", syscall.MS_BIND|syscall.MS_REC, "")
	if err != nil {
		log.Fatal("Failed to bind mount new root: ", err)
	}
	fmt.Println("New root bound to itself successfully")

	// For pivot_root, we need a place to put the old root
	// This directory must be inside the new root
	oldRootPath := filepath.Join(newRootPath, ".old_root")
	fmt.Println("Creating directory for old root:", oldRootPath)
	err = os.MkdirAll(oldRootPath, 0700)
	if err != nil {
		log.Fatal("Failed to create directory for old root: ", err)
	}

	// Now we can use pivot_root to change the root filesystem
	// This moves the current root to oldRootPath and makes newRootPath the new root
	fmt.Println("Executing pivot_root...")
	err = syscall.PivotRoot(newRootPath, oldRootPath)
	if err != nil {
		log.Fatal("Failed to pivot root: ", err)
	}
	fmt.Println("Root pivot completed successfully")

	// Change the current working directory to the new root
	// This is important for relative paths to work correctly
	fmt.Println("Changing working directory to new root...")
	err = syscall.Chdir("/")
	if err != nil {
		log.Fatal("Failed to change working directory to new root: ", err)
	}
	fmt.Println("Working directory changed to new root")
	
	// Note: We could also unmount the old root here to completely hide the host filesystem,
	// but in some container implementations they leave it mounted for debugging purposes
	// Or in some cases, unmount it later once everything else is set up
}