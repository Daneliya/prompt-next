//go:build darwin && cgo

#import <AVFoundation/AVFoundation.h>
#import <ImageIO/ImageIO.h>
#include <math.h>
#include <stdlib.h>
#include <string.h>

void *promptVideoPoster(const char *path, int *length) {
    @autoreleasepool {
        NSURL *url = [NSURL fileURLWithFileSystemRepresentation:path isDirectory:NO relativeToURL:nil];
        AVURLAsset *asset = [AVURLAsset URLAssetWithURL:url options:nil];
        AVAssetImageGenerator *generator = [AVAssetImageGenerator assetImageGeneratorWithAsset:asset];
        generator.appliesPreferredTrackTransform = YES;
        generator.maximumSize = CGSizeMake(480, 480);
        generator.requestedTimeToleranceBefore = CMTimeMakeWithSeconds(0.15, 600);
        generator.requestedTimeToleranceAfter = CMTimeMakeWithSeconds(0.15, 600);

        double duration = CMTimeGetSeconds(asset.duration);
        double sample = isfinite(duration) && duration > 0 ? fmin(1.5, duration / 4) : 0;
        CGImageRef image = [generator copyCGImageAtTime:CMTimeMakeWithSeconds(sample, 600) actualTime:NULL error:NULL];
        if (image == NULL && sample != 0) {
            image = [generator copyCGImageAtTime:kCMTimeZero actualTime:NULL error:NULL];
        }
        if (image == NULL) {
            return NULL;
        }

        NSMutableData *jpeg = [NSMutableData data];
        CGImageDestinationRef destination = CGImageDestinationCreateWithData((__bridge CFMutableDataRef)jpeg, CFSTR("public.jpeg"), 1, NULL);
        BOOL saved = NO;
        if (destination != NULL) {
            NSDictionary *options = @{(__bridge NSString *)kCGImageDestinationLossyCompressionQuality: @0.8};
            CGImageDestinationAddImage(destination, image, (__bridge CFDictionaryRef)options);
            saved = CGImageDestinationFinalize(destination);
            CFRelease(destination);
        }
        CGImageRelease(image);
        if (!saved || jpeg.length == 0) {
            return NULL;
        }

        void *bytes = malloc(jpeg.length);
        if (bytes != NULL) {
            memcpy(bytes, jpeg.bytes, jpeg.length);
            *length = (int)jpeg.length;
        }
        return bytes;
    }
}
