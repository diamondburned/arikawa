## What are the performance impacts of this library?

Not a lot for a Discord bot:

# THIS IS OUTDATED. TODO: UPDATE.

```
# Cold functions, or functions that are called once in runtime:
BenchmarkConstructor-8               	  150537	      7617 ns/op
BenchmarkSubcommandConstructor-8     	  155068	      7721 ns/op

# Hot functions, or functions that can be called multiple times:
BenchmarkCall-8                      	 1000000	      1194 ns/op
BenchmarkHelp-8                      	 1751619	       680 ns/op

# Hot functions, but called implicitly on non-message-create events:
BenchmarkReflectChannelID_1Level-8   	10111023	       113 ns/op
BenchmarkReflectChannelID_5Level-8   	 1872080	       686 ns/op
```
