#include <sys/stat.h>

int main(int argc, char *argv[]) {
	if (argc > 1) {
		return mkfifo(argv[1], S_IRWXO);
	}
	return 1;
}
