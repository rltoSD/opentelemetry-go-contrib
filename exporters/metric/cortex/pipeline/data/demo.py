# import random

# f = open("demo.csv", "w")

# for i in range(30000):
#     ictr_val = i * 2
#     ivrec_val = 50 * i if i % 10 == 0 else 0
#     randval = random.randint(-50 * i, 50 * i)
#     f.write(f"ictr,{ictr_val},\"name1, descr1, key1, value1\"\n")
#     f.write(f"ivrec,{ivrec_val},\"name2, descr1, key2, value2\"\n")
#     f.write(f"iudctr,{randval},\"name3, descr3, key3, value3\"\n")

# f.close()

f = open("data/test.csv", "w")

for i in range(100):
    val = i
    multiplier = 1 if i % 2 == 0 else -1

    # f.write(f"ictr,{val},\"name1, descr1, key1, value1\"\n")
    # f.write(f"fctr,{val * 2},\"name1, descr1, key1, value1\"\n")
    # f.write(f"ivrec,{val * 3},\"name1, descr1, key1, value1\"\n")
    # f.write(f"fvrec,{val * 4},\"name1, descr1, key1, value1\"\n")
    # f.write(f"iudctr,{val * 5 * multiplier},\"name1, descr1, key1, value1\"\n")
    # f.write(f"fudctr,{val * 6 * multiplier},\"name1, descr1, key1, value1\"\n")
    # f.write(f"isobs,{val*7},\"name1, descr1, key1, value1\"\n")
    f.write(f"fsobs,{val*8},\"name1, descr1, key1, value1\"\n")
    # f.write(f"ivobs,{val*9},\"name1, descr1, key1, value1\"\n")
    # f.write(f"fvobs,{val*10},\"name1, descr1, key1, value1\"\n")
    # f.write(f"iudobs,{val*11},\"name1, descr1, key1, value1\"\n")
    # f.write(f"fudobs,{val*12},\"name1, descr1, key1, value1\"\n")
f.close()

# 0 + 1 + 2 + 3 + 4 + 5 + 6 + 7 + 8 + 9 = 45
# 0 + 2 + 4 + 6 + 8 + 10 + 12 + 14 + 16 + 18 = 90
# 0 + -1 + 2 + -3 + 4 + -5 + 6 + -7 + 8 + -9 = -5

# ictr 45
# fctr 90
# iudctr -25
# fudctr -30
# ivrec 135
# fvrec 180


# f = open("data/test2.csv", "w")

# for i in range(10):
#     val = i

#     # f.write(f"ictr,{val},\"name1, descr1, key1, value1\"\n")
#     # f.write(f"fctr,{val * 2},\"name1, descr1, key1, value1\"\n")
#     f.write(f"ivrec,{val * 3},\"name1, descr1, key1, value1\"\n")
#     f.write(f"fvrec,{val * 4},\"name1, descr1, key1, value1\"\n")
#     # f.write(f"iudctr,{val * 5 * multiplier},\"name1, descr1, key1, value1\"\n")
#     # f.write(f"fudctr,{val * 6 * multiplier},\"name1, descr1, key1, value1\"\n")
# f.close()