ok, msg = pcall(error, "test", 1)
assert(not ok and msg == "test")
ok, msg = pcall(error, "test", 2)
assert(not ok and msg == "testdata/error.lua:3: test")
ok, msg = pcall(error, "test", 3)
assert(not ok and msg == "test")