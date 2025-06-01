package interceptor

import (
	"context"
	"fmt"
	"regexp"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	grpcmetadata "google.golang.org/grpc/metadata"
)

func UnaryRequestLogger(logger *logrus.Entry) grpc.UnaryServerInterceptor {
	return logging.UnaryServerInterceptor(requestLogger(logger))
}

func UnaryResponseLogger(logger *logrus.Entry) grpc.UnaryServerInterceptor {
	return logging.UnaryServerInterceptor(responseLogger(logger))
}

func StreamRequestLogger(logger *logrus.Entry) grpc.StreamServerInterceptor {
	return logging.StreamServerInterceptor(requestLogger(logger))
}

func StreamResponseLogger(logger *logrus.Entry) grpc.StreamServerInterceptor {
	return logging.StreamServerInterceptor(responseLogger(logger))
}

func requestLogger(logger *logrus.Entry) (logging.Logger, logging.Option) {
	return interceptorLogger(logger.Dup(), grpcmetadata.FromIncomingContext),
		logging.WithLogOnEvents(logging.StartCall)
}

func responseLogger(logger *logrus.Entry) (logging.Logger, logging.Option) {
	return interceptorLogger(logger, grpcmetadata.FromOutgoingContext),
		logging.WithLogOnEvents(logging.FinishCall)
}

var metadataKeyPattern = regexp.MustCompile(`^(olm-.+|user-agent)$`)

func interceptorLogger(l *logrus.Entry, metadataFunc func(ctx context.Context) (grpcmetadata.MD, bool)) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		f := make(map[string]any, len(fields)/2)

		if metadataFields, ok := metadataFunc(ctx); ok {
			for k, v := range metadataFields {
				if !metadataKeyPattern.MatchString(k) {
					continue
				}
				f[k] = v
			}
		}

		i := logging.Fields(fields).Iterator()
		for i.Next() {
			k, v := i.At()
			f[k] = v
		}
		l := l.WithFields(f)

		switch lvl {
		case logging.LevelDebug:
			l.Debug(msg)
		case logging.LevelInfo:
			l.Info(msg)
		case logging.LevelWarn:
			l.Warn(msg)
		case logging.LevelError:
			l.Error(msg)
		default:
			panic(fmt.Sprintf("unknown level %v", lvl))
		}
	})
}
