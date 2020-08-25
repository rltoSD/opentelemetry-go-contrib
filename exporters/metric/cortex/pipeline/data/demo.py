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

f = open("test.csv", "w")

for i in range(10):
    val = i
    f.write(f"ictr,{val},\"name1, descr1, key1, value1\"\n")
    f.write(f"fctr,{val * 2},\"name1, descr1, key1, value1\"\n")
    f.write(f"iudctr,{val * 3},\"name1, descr1, key1, value1\"\n")
    f.write(f"fudctr,{val * 4},\"name1, descr1, key1, value1\"\n")
    f.write(f"ivrec,{val * 5},\"name1, descr1, key1, value1\"\n")
    f.write(f"fvrec,{val * 6},\"name1, descr1, key1, value1\"\n")
f.close()
