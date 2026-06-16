package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"

	"bitbucket-cli/bitbucket"
	"bitbucket-cli/output"

	"github.com/spf13/cobra"
)

// prCmd is the parent for pull request subcommands. The write subcommand
// (comment) registers itself onto this from pr_comment.go.
var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Pull request commands",
}

func init() {
	var (
		listState string
		listLimit int
	)
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List pull requests for a repository",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, client, err := newClient()
			if err != nil {
				return fail(err)
			}
			_, base, err := resolveRepo(cfg)
			if err != nil {
				return fail(err)
			}

			q := url.Values{"pagelen": {fmt.Sprint(bitbucket.DefaultPageLen)}}
			if listState != "" {
				q.Set("state", listState)
			}
			path := fmt.Sprintf("%s/pullrequests?%s", base, q.Encode())

			values, err := client.Paginate(ctx(cmd), path, listLimit, bitbucket.DefaultMaxPages)
			if err != nil {
				return fail(err)
			}
			if err := emitList(values, output.PullRequestSummary, "No pull requests found."); err != nil {
				return fail(err)
			}
			return nil
		},
	}
	listCmd.Flags().StringVar(&listState, "state", "", "Filter by state: OPEN, MERGED, DECLINED, or SUPERSEDED")
	listCmd.Flags().IntVar(&listLimit, "limit", bitbucket.DefaultLimit, "Maximum pull requests to return")

	getCmd := &cobra.Command{
		Use:   "get <id>",
		Short: "Get a single pull request by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return fail(err)
			}
			cfg, client, err := newClient()
			if err != nil {
				return fail(err)
			}
			_, base, err := resolveRepo(cfg)
			if err != nil {
				return fail(err)
			}

			var raw json.RawMessage
			path := fmt.Sprintf("%s/pullrequests/%d", base, id)
			if err := client.Request(ctx(cmd), path, bitbucket.RequestOptions{}, &raw); err != nil {
				return fail(err)
			}
			if err := emitObject(raw, output.PullRequestSummary); err != nil {
				return fail(err)
			}
			return nil
		},
	}

	var commentsLimit int
	commentsCmd := &cobra.Command{
		Use:   "comments <id>",
		Short: "List comments on a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return fail(err)
			}
			cfg, client, err := newClient()
			if err != nil {
				return fail(err)
			}
			_, base, err := resolveRepo(cfg)
			if err != nil {
				return fail(err)
			}

			path := fmt.Sprintf("%s/pullrequests/%d/comments?pagelen=%d", base, id, bitbucket.DefaultPageLen)
			values, err := client.Paginate(ctx(cmd), path, commentsLimit, bitbucket.DefaultMaxPages)
			if err != nil {
				return fail(err)
			}
			if err := emitList(values, output.CommentSummary, "No comments found."); err != nil {
				return fail(err)
			}
			return nil
		},
	}
	commentsCmd.Flags().IntVar(&commentsLimit, "limit", bitbucket.DefaultLimit, "Maximum comments to return")

	var commitsLimit int
	commitsCmd := &cobra.Command{
		Use:   "commits <id>",
		Short: "List commits on a pull request",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := parseID(args[0])
			if err != nil {
				return fail(err)
			}
			cfg, client, err := newClient()
			if err != nil {
				return fail(err)
			}
			_, base, err := resolveRepo(cfg)
			if err != nil {
				return fail(err)
			}

			path := fmt.Sprintf("%s/pullrequests/%d/commits?pagelen=%d", base, id, bitbucket.DefaultPageLen)
			values, err := client.Paginate(ctx(cmd), path, commitsLimit, bitbucket.DefaultMaxPages)
			if err != nil {
				return fail(err)
			}
			if err := emitList(values, output.CommitSummary, "No commits found."); err != nil {
				return fail(err)
			}
			return nil
		},
	}
	commitsCmd.Flags().IntVar(&commitsLimit, "limit", bitbucket.DefaultLimit, "Maximum commits to return")

	prCmd.AddCommand(listCmd, getCmd, commentsCmd, commitsCmd)
	rootCmd.AddCommand(prCmd)
}
