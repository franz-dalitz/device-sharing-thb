package data

type IDGen struct {
	id int
}

func (idg *IDGen) Generate() int {
	id := idg.id
	idg.id++
	return id
}
