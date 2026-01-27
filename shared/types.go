package shared

type TaskCreateArgs struct {
	Goal        string
	AnswerSpec  string
	ContextSpec string
}

type TaskFinishedArgs struct {
	Answer      string
	ContextSpec string
	ContextID   []int

	ErrorInfo string
}

type ContextItem struct {
	ID   int
	Desc string
}
