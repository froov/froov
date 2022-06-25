package compile

import (
	"os"
	"testing"

	cp "github.com/otiai10/copy"
)

func Test_one(t *testing.T) {
	os.RemoveAll("/Users/jimhurd/dev/froov/pawpaw/docs2")
	cp.Copy("/Users/jimhurd/dev/froov/pawpaw/docs", "/Users/jimhurd/dev/froov/pawpaw/docs2")
	RenameFolder("/Users/jimhurd/dev/froov/pawpaw/docs", "/Users/jimhurd/dev/froov/pawpaw/docs2")
}
