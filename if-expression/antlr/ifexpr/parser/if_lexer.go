// Code generated from If.g4 by ANTLR 4.13.2. DO NOT EDIT.

package parser

import (
	"fmt"
	"github.com/antlr4-go/antlr/v4"
	"sync"
	"unicode"
)

// Suppress unused import error
var _ = fmt.Printf
var _ = sync.Once{}
var _ = unicode.IsLetter

type IfLexer struct {
	*antlr.BaseLexer
	channelNames []string
	modeNames    []string
	// TODO: EOF string
}

var IfLexerLexerStaticData struct {
	once                   sync.Once
	serializedATN          []int32
	ChannelNames           []string
	ModeNames              []string
	LiteralNames           []string
	SymbolicNames          []string
	RuleNames              []string
	PredictionContextCache *antlr.PredictionContextCache
	atn                    *antlr.ATN
	decisionToDFA          []*antlr.DFA
}

func iflexerLexerInit() {
	staticData := &IfLexerLexerStaticData
	staticData.ChannelNames = []string{
		"DEFAULT_TOKEN_CHANNEL", "HIDDEN",
	}
	staticData.ModeNames = []string{
		"DEFAULT_MODE",
	}
	staticData.LiteralNames = []string{
		"", "'if'", "'!='", "'=='", "'('", "')'", "'true'", "'false'", "'nil'",
		"", "", "", "", "", "", "'.'",
	}
	staticData.SymbolicNames = []string{
		"", "", "", "", "", "", "", "", "", "DECIMAL_NUMBER", "DOUBLE_STRING",
		"SINGLE_STRING", "PATH", "STRING", "WS", "DOT",
	}
	staticData.RuleNames = []string{
		"T__0", "T__1", "T__2", "T__3", "T__4", "T__5", "T__6", "T__7", "DECIMAL_NUMBER",
		"DOUBLE_STRING", "SINGLE_STRING", "PATH", "STRING", "WS", "DOT", "DecimalDigit",
		"DecimalIntegerLiteral",
	}
	staticData.PredictionContextCache = antlr.NewPredictionContextCache()
	staticData.serializedATN = []int32{
		4, 0, 15, 129, 6, -1, 2, 0, 7, 0, 2, 1, 7, 1, 2, 2, 7, 2, 2, 3, 7, 3, 2,
		4, 7, 4, 2, 5, 7, 5, 2, 6, 7, 6, 2, 7, 7, 7, 2, 8, 7, 8, 2, 9, 7, 9, 2,
		10, 7, 10, 2, 11, 7, 11, 2, 12, 7, 12, 2, 13, 7, 13, 2, 14, 7, 14, 2, 15,
		7, 15, 2, 16, 7, 16, 1, 0, 1, 0, 1, 0, 1, 1, 1, 1, 1, 1, 1, 2, 1, 2, 1,
		2, 1, 3, 1, 3, 1, 4, 1, 4, 1, 5, 1, 5, 1, 5, 1, 5, 1, 5, 1, 6, 1, 6, 1,
		6, 1, 6, 1, 6, 1, 6, 1, 7, 1, 7, 1, 7, 1, 7, 1, 8, 1, 8, 1, 8, 1, 8, 5,
		8, 68, 8, 8, 10, 8, 12, 8, 71, 9, 8, 1, 8, 1, 8, 4, 8, 75, 8, 8, 11, 8,
		12, 8, 76, 3, 8, 79, 8, 8, 1, 9, 1, 9, 5, 9, 83, 8, 9, 10, 9, 12, 9, 86,
		9, 9, 1, 9, 1, 9, 1, 10, 1, 10, 5, 10, 92, 8, 10, 10, 10, 12, 10, 95, 9,
		10, 1, 10, 1, 10, 1, 11, 1, 11, 5, 11, 101, 8, 11, 10, 11, 12, 11, 104,
		9, 11, 4, 11, 106, 8, 11, 11, 11, 12, 11, 107, 1, 12, 1, 12, 1, 13, 1,
		13, 1, 13, 1, 13, 1, 14, 1, 14, 1, 15, 1, 15, 1, 16, 1, 16, 1, 16, 5, 16,
		123, 8, 16, 10, 16, 12, 16, 126, 9, 16, 3, 16, 128, 8, 16, 0, 0, 17, 1,
		1, 3, 2, 5, 3, 7, 4, 9, 5, 11, 6, 13, 7, 15, 8, 17, 9, 19, 10, 21, 11,
		23, 12, 25, 13, 27, 14, 29, 15, 31, 0, 33, 0, 1, 0, 7, 3, 0, 10, 10, 13,
		13, 34, 34, 3, 0, 10, 10, 13, 13, 39, 39, 5, 0, 45, 45, 48, 57, 65, 90,
		95, 95, 97, 122, 4, 0, 48, 57, 65, 90, 95, 95, 97, 122, 3, 0, 9, 10, 13,
		13, 32, 32, 1, 0, 48, 57, 1, 0, 49, 57, 136, 0, 1, 1, 0, 0, 0, 0, 3, 1,
		0, 0, 0, 0, 5, 1, 0, 0, 0, 0, 7, 1, 0, 0, 0, 0, 9, 1, 0, 0, 0, 0, 11, 1,
		0, 0, 0, 0, 13, 1, 0, 0, 0, 0, 15, 1, 0, 0, 0, 0, 17, 1, 0, 0, 0, 0, 19,
		1, 0, 0, 0, 0, 21, 1, 0, 0, 0, 0, 23, 1, 0, 0, 0, 0, 25, 1, 0, 0, 0, 0,
		27, 1, 0, 0, 0, 0, 29, 1, 0, 0, 0, 1, 35, 1, 0, 0, 0, 3, 38, 1, 0, 0, 0,
		5, 41, 1, 0, 0, 0, 7, 44, 1, 0, 0, 0, 9, 46, 1, 0, 0, 0, 11, 48, 1, 0,
		0, 0, 13, 53, 1, 0, 0, 0, 15, 59, 1, 0, 0, 0, 17, 78, 1, 0, 0, 0, 19, 80,
		1, 0, 0, 0, 21, 89, 1, 0, 0, 0, 23, 105, 1, 0, 0, 0, 25, 109, 1, 0, 0,
		0, 27, 111, 1, 0, 0, 0, 29, 115, 1, 0, 0, 0, 31, 117, 1, 0, 0, 0, 33, 127,
		1, 0, 0, 0, 35, 36, 5, 105, 0, 0, 36, 37, 5, 102, 0, 0, 37, 2, 1, 0, 0,
		0, 38, 39, 5, 33, 0, 0, 39, 40, 5, 61, 0, 0, 40, 4, 1, 0, 0, 0, 41, 42,
		5, 61, 0, 0, 42, 43, 5, 61, 0, 0, 43, 6, 1, 0, 0, 0, 44, 45, 5, 40, 0,
		0, 45, 8, 1, 0, 0, 0, 46, 47, 5, 41, 0, 0, 47, 10, 1, 0, 0, 0, 48, 49,
		5, 116, 0, 0, 49, 50, 5, 114, 0, 0, 50, 51, 5, 117, 0, 0, 51, 52, 5, 101,
		0, 0, 52, 12, 1, 0, 0, 0, 53, 54, 5, 102, 0, 0, 54, 55, 5, 97, 0, 0, 55,
		56, 5, 108, 0, 0, 56, 57, 5, 115, 0, 0, 57, 58, 5, 101, 0, 0, 58, 14, 1,
		0, 0, 0, 59, 60, 5, 110, 0, 0, 60, 61, 5, 105, 0, 0, 61, 62, 5, 108, 0,
		0, 62, 16, 1, 0, 0, 0, 63, 79, 3, 33, 16, 0, 64, 65, 3, 33, 16, 0, 65,
		69, 5, 46, 0, 0, 66, 68, 3, 31, 15, 0, 67, 66, 1, 0, 0, 0, 68, 71, 1, 0,
		0, 0, 69, 67, 1, 0, 0, 0, 69, 70, 1, 0, 0, 0, 70, 79, 1, 0, 0, 0, 71, 69,
		1, 0, 0, 0, 72, 74, 5, 46, 0, 0, 73, 75, 3, 31, 15, 0, 74, 73, 1, 0, 0,
		0, 75, 76, 1, 0, 0, 0, 76, 74, 1, 0, 0, 0, 76, 77, 1, 0, 0, 0, 77, 79,
		1, 0, 0, 0, 78, 63, 1, 0, 0, 0, 78, 64, 1, 0, 0, 0, 78, 72, 1, 0, 0, 0,
		79, 18, 1, 0, 0, 0, 80, 84, 5, 34, 0, 0, 81, 83, 8, 0, 0, 0, 82, 81, 1,
		0, 0, 0, 83, 86, 1, 0, 0, 0, 84, 82, 1, 0, 0, 0, 84, 85, 1, 0, 0, 0, 85,
		87, 1, 0, 0, 0, 86, 84, 1, 0, 0, 0, 87, 88, 5, 34, 0, 0, 88, 20, 1, 0,
		0, 0, 89, 93, 5, 39, 0, 0, 90, 92, 8, 1, 0, 0, 91, 90, 1, 0, 0, 0, 92,
		95, 1, 0, 0, 0, 93, 91, 1, 0, 0, 0, 93, 94, 1, 0, 0, 0, 94, 96, 1, 0, 0,
		0, 95, 93, 1, 0, 0, 0, 96, 97, 5, 39, 0, 0, 97, 22, 1, 0, 0, 0, 98, 102,
		5, 46, 0, 0, 99, 101, 7, 2, 0, 0, 100, 99, 1, 0, 0, 0, 101, 104, 1, 0,
		0, 0, 102, 100, 1, 0, 0, 0, 102, 103, 1, 0, 0, 0, 103, 106, 1, 0, 0, 0,
		104, 102, 1, 0, 0, 0, 105, 98, 1, 0, 0, 0, 106, 107, 1, 0, 0, 0, 107, 105,
		1, 0, 0, 0, 107, 108, 1, 0, 0, 0, 108, 24, 1, 0, 0, 0, 109, 110, 7, 3,
		0, 0, 110, 26, 1, 0, 0, 0, 111, 112, 7, 4, 0, 0, 112, 113, 1, 0, 0, 0,
		113, 114, 6, 13, 0, 0, 114, 28, 1, 0, 0, 0, 115, 116, 5, 46, 0, 0, 116,
		30, 1, 0, 0, 0, 117, 118, 7, 5, 0, 0, 118, 32, 1, 0, 0, 0, 119, 128, 5,
		48, 0, 0, 120, 124, 7, 6, 0, 0, 121, 123, 3, 31, 15, 0, 122, 121, 1, 0,
		0, 0, 123, 126, 1, 0, 0, 0, 124, 122, 1, 0, 0, 0, 124, 125, 1, 0, 0, 0,
		125, 128, 1, 0, 0, 0, 126, 124, 1, 0, 0, 0, 127, 119, 1, 0, 0, 0, 127,
		120, 1, 0, 0, 0, 128, 34, 1, 0, 0, 0, 10, 0, 69, 76, 78, 84, 93, 102, 107,
		124, 127, 1, 6, 0, 0,
	}
	deserializer := antlr.NewATNDeserializer(nil)
	staticData.atn = deserializer.Deserialize(staticData.serializedATN)
	atn := staticData.atn
	staticData.decisionToDFA = make([]*antlr.DFA, len(atn.DecisionToState))
	decisionToDFA := staticData.decisionToDFA
	for index, state := range atn.DecisionToState {
		decisionToDFA[index] = antlr.NewDFA(state, index)
	}
}

// IfLexerInit initializes any static state used to implement IfLexer. By default the
// static state used to implement the lexer is lazily initialized during the first call to
// NewIfLexer(). You can call this function if you wish to initialize the static state ahead
// of time.
func IfLexerInit() {
	staticData := &IfLexerLexerStaticData
	staticData.once.Do(iflexerLexerInit)
}

// NewIfLexer produces a new lexer instance for the optional input antlr.CharStream.
func NewIfLexer(input antlr.CharStream) *IfLexer {
	IfLexerInit()
	l := new(IfLexer)
	l.BaseLexer = antlr.NewBaseLexer(input)
	staticData := &IfLexerLexerStaticData
	l.Interpreter = antlr.NewLexerATNSimulator(l, staticData.atn, staticData.decisionToDFA, staticData.PredictionContextCache)
	l.channelNames = staticData.ChannelNames
	l.modeNames = staticData.ModeNames
	l.RuleNames = staticData.RuleNames
	l.LiteralNames = staticData.LiteralNames
	l.SymbolicNames = staticData.SymbolicNames
	l.GrammarFileName = "If.g4"
	// TODO: l.EOF = antlr.TokenEOF

	return l
}

// IfLexer tokens.
const (
	IfLexerT__0           = 1
	IfLexerT__1           = 2
	IfLexerT__2           = 3
	IfLexerT__3           = 4
	IfLexerT__4           = 5
	IfLexerT__5           = 6
	IfLexerT__6           = 7
	IfLexerT__7           = 8
	IfLexerDECIMAL_NUMBER = 9
	IfLexerDOUBLE_STRING  = 10
	IfLexerSINGLE_STRING  = 11
	IfLexerPATH           = 12
	IfLexerSTRING         = 13
	IfLexerWS             = 14
	IfLexerDOT            = 15
)
