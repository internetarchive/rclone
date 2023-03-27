package v2

import (
	"os"
	"testing"

	"github.com/rclone/rclone/backend/vault"
	"github.com/rclone/rclone/fstest/fstests"
)

func TestIntegration(t *testing.T) {
	var remoteName string
	if v := os.Getenv("VAULT_TEST_REMOTE_NAME"); v != "" {
		remoteName = v
	} else {
		t.Skip("VAULT_TEST_REMOTE_NAME env not set, skipping")
	}
	fstests.Run(t, &fstests.Opt{
		RemoteName:               remoteName,
		NilObject:                (*vault.Object)(nil),
		SkipFsCheckWrap:          true,
		SkipInvalidUTF8:          true,
		SkipBadWindowsCharacters: true,
		QuickTestOK:              true,
	})
	// --- FAIL: TestIntegration (136.80s)
	//     --- SKIP: TestIntegration/FsCheckWrap (0.00s)
	//     --- PASS: TestIntegration/FsCommand (0.00s)
	//     --- PASS: TestIntegration/FsRmdirNotFound (0.54s)
	//     --- PASS: TestIntegration/FsString (0.00s)
	//     --- PASS: TestIntegration/FsName (0.00s)
	//     --- PASS: TestIntegration/FsRoot (0.00s)
	//     --- FAIL: TestIntegration/FsRmdirEmpty (0.08s)
	//     --- FAIL: TestIntegration/FsMkdir (133.00s)
	//         --- PASS: TestIntegration/FsMkdir/FsMkdirRmdirSubdir (6.66s)
	//         --- PASS: TestIntegration/FsMkdir/FsListEmpty (0.28s)
	//         --- PASS: TestIntegration/FsMkdir/FsListDirEmpty (0.28s)
	//         --- SKIP: TestIntegration/FsMkdir/FsListRDirEmpty (0.00s)
	//         --- PASS: TestIntegration/FsMkdir/FsListDirNotFound (0.29s)
	//         --- SKIP: TestIntegration/FsMkdir/FsListRDirNotFound (0.00s)
	//         --- FAIL: TestIntegration/FsMkdir/FsEncoding (105.73s)
	//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/control_chars (6.16s)
	//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/dot (6.14s)
	//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/dot_dot (6.20s)
	//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/punctuation (0.70s)
	//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_space (6.17s)
	//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_tilde (6.14s)
	//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_CR (6.15s)
	//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_LF (6.26s)
	//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_HT (6.02s)
	//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_VT (6.45s)
	//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/leading_dot (6.20s)
	//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_space (5.84s)
	//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_CR (6.28s)
	//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_LF (6.01s)
	//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_HT (6.18s)
	//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_VT (6.15s)
	//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/trailing_dot (6.17s)
	//             --- SKIP: TestIntegration/FsMkdir/FsEncoding/invalid_UTF-8 (0.00s)
	//             --- FAIL: TestIntegration/FsMkdir/FsEncoding/URL_encoding (6.23s)
	//         --- PASS: TestIntegration/FsMkdir/FsNewObjectNotFound (0.50s)
	//         --- FAIL: TestIntegration/FsMkdir/FsPutError (0.00s)
	//         --- FAIL: TestIntegration/FsMkdir/FsPutZeroLength (5.50s)
	//         --- SKIP: TestIntegration/FsMkdir/FsOpenWriterAt (0.00s)
	//         --- SKIP: TestIntegration/FsMkdir/FsChangeNotify (0.00s)
	//         --- FAIL: TestIntegration/FsMkdir/FsPutFiles (5.44s)
	//         --- SKIP: TestIntegration/FsMkdir/FsPutChunked (0.00s)
	//         --- FAIL: TestIntegration/FsMkdir/FsUploadUnknownSize (5.87s)
	//             --- FAIL: TestIntegration/FsMkdir/FsUploadUnknownSize/FsPutUnknownSize (0.40s)
	//             --- FAIL: TestIntegration/FsMkdir/FsUploadUnknownSize/FsUpdateUnknownSize (5.46s)
	//         --- PASS: TestIntegration/FsMkdir/FsRootCollapse (1.85s)
	//     --- FAIL: TestIntegration/FsShutdown (0.96s)
	// FAIL
	// FAIL    command-line-arguments  136.812s
	// FAIL

}
