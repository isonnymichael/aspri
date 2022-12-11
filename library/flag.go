package library

import "flag"

/** Flag Struct */
type Flag struct {
	/** Mode */
	Docker                    *bool
	Git                       *bool
	DockerComposeRestart      *bool
	QuoteofTheDay             *bool
	SearchandReplaceDirectory *bool
	WPPluginBuildCheck        *bool
	WPRefactor                *bool

	/** Additional Mode */
	Prune *bool
	Push  *bool

	/** Parameters */
	ID      *string
	From    *string
	Message *string
	Path    *string
	To      *string
}

/** Get Flag */
func GetFlag() Flag {
	flags := Flag{}

	/** Mode */
	flags.Docker = flag.Bool("docker", false, "Docker Mode")
	flags.DockerComposeRestart = flag.Bool("docker-compose-restart", false, "Docker Compose Restart")
	flags.Git = flag.Bool("git", false, "Git Mode")
	flags.WPPluginBuildCheck = flag.Bool("wp-plugin-build-check", false, "WP Check Plugin Comply with Directory")
	flags.WPRefactor = flag.Bool("wp-refactor", false, "Refactor Library")
	flags.QuoteofTheDay = flag.Bool("quote-of-the-day", false, "show quote of the day")
	flags.SearchandReplaceDirectory = flag.Bool("search-replace-directory", false, "do search and replace")

	/** Parameters */
	flags.ID = flag.String("id", "", "Identifier (Docker Mode): Container")
	flags.Prune = flag.Bool("prune", false, "Prune (Docker Mode): Container")
	flags.Push = flag.Bool("push", false, "Push (Git Mode): Commit and Push")
	flags.Message = flag.String("m", "", "Message (Git Mode): Commit Message")

	/** Refactor */
	flags.Path = flag.String("path", "", "Refactor : Path to Directory")
	flags.From = flag.String("from", "", "Refactor Text From")
	flags.To = flag.String("to", "", "Refactor Text To")

	return flags
}
