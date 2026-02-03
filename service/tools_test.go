package service_test

import (
	"fmt"
	"multi-agent/service"
	"testing"
	_ "multi-agent/shared"
)

func TestExtractAllBinaries(t *testing.T) {
	t.Run("test extract binary", func(t *testing.T) {
		testCases := []string{
			"cat aa.txt | grep hello",
			`ls -l`,
			`mkdir -p test && touch test/file.txt`,
			`curl -s https://example.com | tidy -iq`,
			`find . -name "*.go" -exec grep "TODO" {} +`,
			`$(which python3) -m http.server`,
			`if ! command -v git &> /dev/null; then echo "no git"; fi`,
			`tail -f /var/log/syslog | awk '{print $5}' > filtered.log`,
		}
		for _, cmd := range testCases {
			got, gotErr := service.ExtractAllBinaries(cmd)
			if gotErr != nil {
				fmt.Printf("gotErr: %v\n", gotErr)
			} else {
				fmt.Printf("got: %v\n", got)
			}

		}
	})
}

func TestBashRun(t *testing.T) {
	t.Run("test extract binary", func(t *testing.T) {
		test := service.BashTool{}
		test.AddRepo("/root/multi-agent")
		testCashs := []string{
			"ls | grep go",
			"pwd",
			`echo "hello you" > test.txt`,
		}
		for _, cmd := range testCashs {
			res, err := test.Run(cmd, "")
			if err != nil {
				fmt.Printf("err: %v\n", err)
				continue
			}
			fmt.Printf("%v\n", res)
		}
	})
}
