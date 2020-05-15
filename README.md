A small proxy which runs in between the main Extraterm application on Windows and insides of WSL Linux environment. The proxy runs inside WSL as a Linux binary and has a single pipe to Extraterm. From here it receives commands to spawn different command line applications (i.e. shells) and to shuffle the terminal stdin/stdout data back and forth between the application and Extraterm itself.

