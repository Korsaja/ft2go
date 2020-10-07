package main

type filter func(rec *ftrecord) bool

func applyFilter(fn filter, rec *ftrecord) bool {
	return fn(rec)
}
// parallel filter

func SliceFilter(fn filter, records []*ftrecord) []*ftrecord {
	var result []*ftrecord
	f := func(batch []*ftrecord,c chan struct{}) {
		for _, rec := range batch {
			if applyFilter(fn,rec){
				result = append(result,rec)
			}
		}
		c <- struct{}{}
	}
	maxGo := 4
	c := make(chan struct{},maxGo)
	length := len(records)
	for i := 0; i < length; i+=maxGo {
		batch := records[i:min(i+maxGo, length)]
		go f(batch,c)
	}

	for i := 0; i < maxGo; i++ {
		<-c
	}
	return result
}

func min(a, b int) int {
	if a <= b {
		return a
	}
	return b
}