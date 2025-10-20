package main

import (
	"fmt"
	"os"

	"github.com/pyenv-win/pyenv-win/internal/commands"
	"github.com/pyenv-win/pyenv-win/internal/config"
)

const version = "4.0.0"

func main() {
	if len(os.Args) < 2 {
		showUsage()
		os.Exit(0)
	}

	cfg := config.New()
	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "install":
		handleInstall(cfg, args)
	case "update":
		handleUpdate(cfg, args)
	case "uninstall":
		handleUninstall(cfg, args)
	case "versions":
		handleVersions(cfg, args)
	case "version":
		handleVersion(cfg, args)
	case "global":
		handleGlobal(cfg, args)
	case "local":
		handleLocal(cfg, args)
	case "version-name", "vname":
		handleVersionName(cfg, args)
	case "latest":
		handleLatest(cfg, args)
	case "--version", "-v":
		fmt.Printf("pyenv %s\n", version)
	case "help", "--help", "-h":
		showHelp()
	case "commands":
		showCommands()
	case "rehash":
		handleRehash(cfg, args)
	case "which":
		handleWhich(cfg, args)
	case "whence":
		handleWhence(cfg, args)
	case "exec":
		handleExec(cfg, args)
	case "shims":
		handleShims(cfg, args)
	case "duplicate":
		handleDuplicate(cfg, args)
	default:
		fmt.Printf("pyenv: no such command `%s'\n", command)
		os.Exit(1)
	}
}

func handleInstall(cfg *config.PyenvConfig, args []string) {
	cmd := commands.NewInstallCommand(cfg)
	opts := &commands.InstallOptions{}

	// Parse arguments
	versions := []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-l", "--list":
			opts.List = true
		case "-a", "--all":
			opts.All = true
		case "-c", "--clear":
			opts.Clear = true
		case "-f", "--force":
			opts.Force = true
		case "-s", "--skip-existing":
			opts.SkipExisting = true
		case "-q", "--quiet":
			opts.Quiet = true
		case "-r", "--register":
			opts.Register = true
		case "--dev":
			opts.Dev = true
		case "--offline":
			opts.Offline = true
		case "--32only":
			opts.Only32 = true
		case "--64only":
			opts.Only64 = true
		case "--help":
			showInstallHelp()
			return
		default:
			versions = append(versions, arg)
		}
	}

	opts.Versions = versions

	if err := cmd.Execute(opts); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleUpdate(cfg *config.PyenvConfig, args []string) {
	cmd := commands.NewUpdateCommand(cfg)

	ignoreErrors := false
	for _, arg := range args {
		if arg == "--ignore" {
			ignoreErrors = true
		} else if arg == "--help" {
			showUpdateHelp()
			return
		}
	}

	if err := cmd.Execute(ignoreErrors); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleUninstall(cfg *config.PyenvConfig, args []string) {
	cmd := commands.NewVersionsCommand(cfg)

	force := false
	var version string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-f", "--force":
			force = true
		case "--help":
			showUninstallHelp()
			return
		default:
			version = arg
		}
	}

	if version == "" {
		fmt.Println("Usage: pyenv uninstall [-f] <version>")
		os.Exit(1)
	}

	if err := cmd.Uninstall(version, force); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleVersions(cfg *config.PyenvConfig, args []string) {
	cmd := commands.NewVersionsCommand(cfg)

	if err := cmd.ListInstalled(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleVersion(cfg *config.PyenvConfig, args []string) {
	cmd := commands.NewVersionsCommand(cfg)

	versions, err := cmd.GetCurrent()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	for _, v := range versions {
		fmt.Println(v)
	}
}

func handleGlobal(cfg *config.PyenvConfig, args []string) {
	cmd := commands.NewVersionsCommand(cfg)

	if len(args) == 0 {
		// Get global version
		versions, err := cmd.GetGlobal()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		for _, v := range versions {
			fmt.Println(v)
		}
	} else {
		// Set global version
		if err := cmd.SetGlobal(args...); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func handleLocal(cfg *config.PyenvConfig, args []string) {
	cmd := commands.NewVersionsCommand(cfg)

	unset := false
	versions := []string{}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--unset" {
			unset = true
		} else {
			versions = append(versions, arg)
		}
	}

	if unset {
		// Unset local version
		if err := cmd.UnsetLocal(""); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else if len(versions) == 0 {
		// Get local version
		localVersions, err := cmd.GetLocal("")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		for _, v := range localVersions {
			fmt.Println(v)
		}
	} else {
		// Set local version
		if err := cmd.SetLocal("", versions...); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func handleVersionName(cfg *config.PyenvConfig, args []string) {
	cmd := commands.NewVersionsCommand(cfg)

	version, err := cmd.GetVersionName()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(version)
}

func handleLatest(cfg *config.PyenvConfig, args []string) {
	cmd := commands.NewVersionsCommand(cfg)

	known := false
	prefix := ""

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "-k" || arg == "--known" {
			known = true
		} else {
			prefix = arg
		}
	}

	if prefix == "" {
		fmt.Println("Usage: pyenv latest [-k|--known] <prefix>")
		os.Exit(1)
	}

	latest, err := cmd.Latest(prefix, known)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(latest)
}

func handleRehash(cfg *config.PyenvConfig, args []string) {
	cmd := commands.NewShimsCommand(cfg)
	if err := cmd.Rehash(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleWhich(cfg *config.PyenvConfig, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: pyenv which <command>")
		os.Exit(1)
	}

	cmd := commands.NewShimsCommand(cfg)
	path, err := cmd.Which(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(path)
}

func handleWhence(cfg *config.PyenvConfig, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: pyenv whence <command>")
		os.Exit(1)
	}

	cmd := commands.NewShimsCommand(cfg)
	versions, err := cmd.Whence(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	for _, v := range versions {
		fmt.Println(v)
	}
}

func handleExec(cfg *config.PyenvConfig, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: pyenv exec <command> [args...]")
		os.Exit(1)
	}

	cmd := commands.NewShimsCommand(cfg)
	if err := cmd.Exec(args[0], args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleShims(cfg *config.PyenvConfig, args []string) {
	cmd := commands.NewShimsCommand(cfg)
	if err := cmd.ListShims(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleDuplicate(cfg *config.PyenvConfig, args []string) {
	// Duplicate would duplicate a version
	fmt.Printf("pyenv: duplicate command not yet implemented\n")
}

func showUsage() {
	fmt.Println("Usage: pyenv <command> [<args>]")
	fmt.Println()
	fmt.Println("Some useful pyenv commands are:")
	fmt.Println("   commands    List all available pyenv commands")
	fmt.Println("   local       Set or show the local application-specific Python version")
	fmt.Println("   global      Set or show the global Python version")
	fmt.Println("   install     Install a Python version")
	fmt.Println("   uninstall   Uninstall a specific Python version")
	fmt.Println("   update      Update the list of available versions")
	fmt.Println("   versions    List all Python versions available to pyenv")
	fmt.Println("   version     Show the current Python version and its origin")
	fmt.Println()
	fmt.Println("See `pyenv help <command>' for information on a specific command.")
}

func showHelp() {
	showUsage()
}

func showCommands() {
	commands := []string{
		"commands",
		"duplicate",
		"exec",
		"global",
		"help",
		"install",
		"latest",
		"local",
		"rehash",
		"shims",
		"uninstall",
		"update",
		"version",
		"version-name",
		"versions",
		"vname",
		"whence",
		"which",
	}

	for _, cmd := range commands {
		fmt.Println(cmd)
	}
}

func showInstallHelp() {
	fmt.Println("Usage: pyenv install [-s] [-f] <version> [<version> ...] [-r|--register]")
	fmt.Println("       pyenv install [-f] [--32only|--64only] -a|--all")
	fmt.Println("       pyenv install [-f] -c|--clear")
	fmt.Println("       pyenv install -l|--list")
	fmt.Println()
	fmt.Println("  -l/--list              List all available versions")
	fmt.Println("  -a/--all               Installs all known version from the local version DB cache")
	fmt.Println("  -c/--clear             Removes downloaded installers from the cache to free space")
	fmt.Println("  -f/--force             Install even if the version appears to be installed already")
	fmt.Println("  -s/--skip-existing     Skip the installation if the version appears to be installed already")
	fmt.Println("  -r/--register          Register version for py launcher")
	fmt.Println("  -q/--quiet             Install using /quiet. This does not show the UI nor does it prompt for inputs")
	fmt.Println("  --32only               Installs only 32bit Python using -a/--all switch, no effect on 32-bit windows.")
	fmt.Println("  --64only               Installs only 64bit Python using -a/--all switch, no effect on 32-bit windows.")
	fmt.Println("  --dev                  Installs precompiled standard libraries, debug symbols, and debug binaries (only applies to web installer).")
	fmt.Println("  --offline              Download Python installers from Nexus3 server instead of internet")
	fmt.Println("  --help                 Help, list of options allowed on pyenv install")
}

func showUpdateHelp() {
	fmt.Println("Usage: pyenv update [--ignore]")
	fmt.Println()
	fmt.Println("  --ignore  Ignores any HTTP/VBScript errors that occur during downloads.")
	fmt.Println()
	fmt.Println("Updates the internal database of python installer URL's.")
}

func showUninstallHelp() {
	fmt.Println("Usage: pyenv uninstall [-f|--force] <version>")
	fmt.Println()
	fmt.Println("  -f/--force    Force uninstall without confirmation")
}
