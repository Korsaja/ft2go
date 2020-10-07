package main

type filter func(rec *ftrecord) bool

func applyFilter(fn filter, rec *ftrecord) bool {
	return fn(rec)
}
// parallel filter

func SliceFilter(fn filter, records []*ftrecord) []*ftrecord {
	var result []*ftrecord
	f := func(i, j int,c chan struct{}) {
		for ; i < j; i++ {
			if applyFilter(fn,records[i]){
				result = append(result,result[i])
			}
		}
		c <- struct{}{}
	}
	maxGo := 4
	c := make(chan struct{},maxGo)
	length := len(records)
	for i := 0; i < maxGo; i++ {
		go f(i*length/maxGo, (i+1)*length/maxGo,c)
	}

	for i := 0; i < maxGo; i++ {
		<-c
	}
	return result
}
