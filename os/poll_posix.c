
#define usbi_write write
#define usbi_read read
#define usbi_close close
#define usbi_poll poll
int usbi_pipe(int pipefd[2]);

int usbi_pipe(int pipefd[2])
{
	int ret = pipe(pipefd);
	if (ret != 0) {
		return ret;
	}
	ret = fcntl(pipefd[1], F_GETFL);
	if (ret == -1) {
		// usbi_dbg("Failed to get pipe fd flags: %d", errno);
		goto err_close_pipe;
	}
	ret = fcntl(pipefd[1], F_SETFL, ret | O_NONBLOCK);
	if (ret != 0) {
		// usbi_dbg("Failed to set non-blocking on new pipe: %d", errno);
		goto err_close_pipe;
	}

	return 0;

err_close_pipe:
	usbi_close(pipefd[0]);
	usbi_close(pipefd[1]);
	return ret;
}
