#include <cstdio>
#include <sys/stat.h>

int main(int argc, char *argv[]) {
	int result = mkfifo("/data/local/tmp/share", S_IRWXO);
	if (result != 0) {
		printf("Error creating pipe");
	}
	return 0;
}
