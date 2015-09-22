#ifndef DUMP978_H
#define DUMP978_H

// $ gcc -c -O2 -g -Wall -Werror -Ifec -fpic -DBUILD_LIB=1 dump978.c fec.c fec/decode_rs_char.c fec/init_rs_char.c
// $ gcc -shared -lm -o ../libdump978.so dump978.o fec.o decode_rs_char.o init_rs_char.o

typedef void (*CallBack)(char updown, uint8_t *data, int len, int rs_errors, int signal_strength);

extern void Dump978Init(CallBack cb);
extern int process_data(char *data, int dlen);

#endif /* DUMP978_H */
