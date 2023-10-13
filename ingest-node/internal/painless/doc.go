package painless

// go:generate curl -L -O https://raw.githubusercontent.com/elastic/elasticsearch/3636d3d6ac492dda2dc2400e104b69319b753daa/modules/lang-painless/src/main/antlr/PainlessLexer.g4
// go:generate curl -L -O https://github.com/elastic/elasticsearch/raw/3636d3d6ac492dda2dc2400e104b69319b753daa/modules/lang-painless/src/main/antlr/PainlessLexer.tokens
// go:generate curl -L -O https://github.com/elastic/elasticsearch/raw/3636d3d6ac492dda2dc2400e104b69319b753daa/modules/lang-painless/src/main/antlr/PainlessParser.g4
// go:generate curl -L -O https://github.com/elastic/elasticsearch/raw/3636d3d6ac492dda2dc2400e104b69319b753daa/modules/lang-painless/src/main/antlr/PainlessParser.tokens

//go:generate rm -rf parser/
//go:generate antlr -Dlanguage=Go -o parser -visitor PainlessLexer.g4
//go:generate antlr -Dlanguage=Go -o parser -visitor PainlessParser.g4
