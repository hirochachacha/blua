package parser_test

import (
	"os"

	"github.com/hirochachacha/plua/compiler/ast/printer"
	"github.com/hirochachacha/plua/compiler/parser"
)

func ExampleParseFile() {
	ast, err := parser.ParseFile("testdata/example.lua", 0)
	if err != nil {
		panic(err)
	}

	printer.FprintTree(os.Stdout, ast)

	// Output:
	// File { ?:1-?:3
	//   Shebang: "#!/usr/bin/env lua"
	//   Chunk: {
	//     ExprStmt { ?:3-?:3
	//       X: CallExpr { ?:3-?:3
	//         X: Name { ?:3-?:3
	//           NamePos: ?:3
	//           Name: print
	//         }
	//         Colon: ?:-1
	//         Name: nil
	//         Lparen: ?:-1
	//         Args: {
	//           BasicLit { ?:3-?:3
	//             Token.Type: STRING
	//             Token.Pos: ?:3
	//             Token.Lit: "Hello World!"
	//           }
	//         }
	//         Rparen: ?:-1
	//       }
	//     }
	//   }
	//   Comments: {
	//   }
	// }
}
