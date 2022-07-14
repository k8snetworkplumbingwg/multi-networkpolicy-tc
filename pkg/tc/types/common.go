package types

// CmdLineGenerator is an interface for generating tc command line args for a tc object
type CmdLineGenerator interface {
	// GenCmdLineArgs returns tc command line arguments which can be incorporated
	// when invoking tc command via shell
	GenCmdLineArgs() []string
}
