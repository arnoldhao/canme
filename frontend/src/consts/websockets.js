export const WS_NAMESPACE = {
    DOWNTASKS: 'downtasks',
    SUBTITLES: 'subtitles'
}

export const WS_REQUEST_EVENT = {
}

export const WS_RESPONSE_EVENT = {
    // DOWNTASKS
    EVENT_DOWNTASKS_PROGRESS: 'response_downtasks_progress',
    EVENT_DOWNTASKS_SIGNAL: 'response_downtasks_signal',
    EVENT_DOWNTASKS_INSTALLING: 'response_downtasks_installing',
    EVENT_DOWNTASKS_COOKIE_SYNC: 'response_downtasks_cookie_sync',
    // SUBTITLE
    EVENT_SUBTITLE_PROGRESS: 'response_subtitle_progress'
}