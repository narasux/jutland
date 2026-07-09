#import <Cocoa/Cocoa.h>
#include "magnify_darwin.h"

static double jutland_magnify_accum = 0.0;
static BOOL   jutland_magnify_inited = NO;

void jutland_magnify_init(void) {
    if (jutland_magnify_inited) return;
    jutland_magnify_inited = YES;

    dispatch_async(dispatch_get_main_queue(), ^{
        [NSEvent addLocalMonitorForEventsMatchingMask:NSEventMaskMagnify handler:^NSEvent *(NSEvent *event) {
            jutland_magnify_accum += [event magnification];
            return event;
        }];
    });
}

double jutland_magnify_poll(void) {
    double v = jutland_magnify_accum;
    jutland_magnify_accum = 0.0;
    return v;
}
