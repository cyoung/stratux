CFLAGS+=-O2 -Wall -Werror -Ifec
LDFLAGS=
LIBS=-lm
CC=gcc

all: dump978 uat2json uat2text uat2esnt extract_nexrad

%.o: %.c *.h
	$(CC) $(CPPFLAGS) $(CFLAGS) -c $< -o $@

lib:
	$(CC) -c -O2 -g -Wall -Werror -Ifec -fpic -DBUILD_LIB=1 dump978.c fec.c fec/decode_rs_char.c fec/init_rs_char.c
	$(CC) -shared -lm -o ../libdump978.so dump978.o fec.o decode_rs_char.o init_rs_char.o 

dump978: dump978.o fec.o fec/decode_rs_char.o fec/init_rs_char.o
	$(CC) -g -o $@ $^ $(LDFLAGS) $(LIBS)

uat2json: uat2json.o uat_decode.o reader.o
	$(CC) -g -o $@ $^ $(LDFLAGS) $(LIBS)

uat2text: uat2text.o uat_decode.o reader.o
	$(CC) -g -o $@ $^ $(LDFLAGS) $(LIBS)

uat2esnt: uat2esnt.o uat_decode.o reader.o
	$(CC) -g -o $@ $^ $(LDFLAGS) $(LIBS)

extract_nexrad: extract_nexrad.o uat_decode.o reader.o
	$(CC) -g -o $@ $^ $(LDFLAGS) $(LIBS)

fec_tests: fec_tests.o fec.o fec/decode_rs_char.o fec/init_rs_char.o
	$(CC) -g -o $@ $^ $(LDFLAGS) $(LIBS)

test: fec_tests
	./fec_tests

clean:
	rm -f *~ *.o fec/*.o dump978 uat2json uat2text uat2esnt fec_tests
