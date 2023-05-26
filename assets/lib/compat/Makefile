TARGET = libcompat.a

$(TARGET): linux64 linux32

linux64_compat.o: glibc_compat.c
	gcc -c $< -o $@

linux32_compat.o: glibc_compat.c
	gcc -m32 -c $< -o $@

linux64: linux64_compat.o
	ar rcs ../$@/$(TARGET) $^

linux32: linux32_compat.o
	ar rcs ../$@/$(TARGET) $^

clean:
	rm -f *.o ../*/$(TARGET)
