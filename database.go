package main

type Database interface {
	Add(string)
	Delete(string)
	List() string
	Store(string, float64)
//	Store(string, int)
//	Store(string, string)
	Load(string) []float64
	Exists(string) bool
}

type Data struct {
	Db map[string][]float64
}

func (d Data) Add(s string) {
	d.Db[s] = make([]float64, 0, 100000)
}

func (d Data) Delete(s string) {
	delete(d.Db, s)
}

func (d Data) List() string {
	return "not implemented"
}

func (d Data) Store(s string, f float64) {
	d.Db[s] = append(d.Db[s], f)
}

func (d Data) Load(s string) []float64 {
	return d.Db[s]
}

func (d Data) Exists(s string) (b bool) {
	_, b = d.Db[s]
	return b
}
