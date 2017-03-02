package tasks

type Task interface {
	Do(uuid string)
}

type UUIDCollector interface {
	Collect() chan string
	Length() int
}

type NativeContentTask struct {
}
