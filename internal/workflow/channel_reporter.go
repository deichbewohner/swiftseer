package workflow

type ChannelReporter struct {
    ch chan<- Event
}

func NewChannelReporter(ch chan<- Event) *ChannelReporter {
	return &ChannelReporter{ch: ch}
}

func (r *ChannelReporter) OnEvent(e Event) { r.ch <- e }
