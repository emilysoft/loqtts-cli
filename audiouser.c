/************************************************************************/
/*                                                                      */
/*        LoqTTS7AudioUser - Loquendo TTS 7 custom audio destination    */
/*                                                                      */
/*   A custom "Audio Destination" DLL for the Loquendo TTS 7 SDK.       */
/*   Instead of writing audio to a file or a board, it accumulates the  */
/*   generated PCM in a memory buffer and, when the stream ends, emits  */
/*   a complete RIFF WAV file to stdout.                                */
/*                                                                      */
/*   Use it with: ttsSetAudio(hReader, "LoqTTS7AudioUser", NULL, rate,  */
/*                 tts_LINEAR, channels, 0);                            */
/*                                                                      */
/*   The DLL must sit in the LTTS7 bin/ directory (where LoqTTS7.dll    */
/*   lives) and export a single entry point named after the DLL.        */
/*                                                                      */
/************************************************************************/

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "loqtts.h"

#if defined(_WIN32)
#include <fcntl.h>
#include <io.h>
#endif

#define OUTOFMEMORY    1
#define INVALIDPARAM   2

typedef struct {
    unsigned char *buf;          /* PCM accumulation buffer          */
    size_t len;                  /* bytes currently in the buffer    */
    size_t cap;                  /* allocated buffer size            */
    unsigned int SampleRate;     /* audio sample rate in Hz          */
    ttsAudioSampleType nChannels;/* tts_MONO or tts_STEREO           */
    ttsAudioEncodingType coding; /* tts_LINEAR / tts_ALAW / tts_ULAW */
} ChannelType;

static void *xrealloc(void *ptr, size_t size)
{
    void *p = realloc(ptr, size);
    if (!p)
    {
        fprintf(stderr, "LoqTTS7AudioUser: out of memory\n");
        exit(1);
    }
    return p;
}

static int ensure_cap(ChannelType *ch, size_t extra)
{
    if (ch->len + extra > ch->cap)
    {
        size_t ncap = ch->cap ? ch->cap : 4096;
        while (ch->len + extra > ncap)
            ncap *= 2;
        ch->buf = xrealloc(ch->buf, ncap);
        ch->cap = ncap;
    }
    return 0;
}

static void emit_wav(ChannelType *ch)
{
    unsigned int sr = ch->SampleRate;
    unsigned int chans = ch->nChannels == tts_MONO ? 1 : 2;
    unsigned int bits = 16;
    unsigned int block = chans * (bits / 8);
    unsigned int byteRate = sr * block;
    unsigned int dataSize = (unsigned int)ch->len;
    unsigned int riffSize = 36 + dataSize;
    unsigned char hdr[44];

    memset(hdr, 0, sizeof(hdr));
    memcpy(hdr + 0, "RIFF", 4);
    hdr[4]  = riffSize & 0xff; hdr[5]  = (riffSize >> 8) & 0xff;
    hdr[6]  = (riffSize >> 16) & 0xff; hdr[7] = (riffSize >> 24) & 0xff;
    memcpy(hdr + 8, "WAVE", 4);
    memcpy(hdr + 12, "fmt ", 4);
    hdr[16] = 16;                          /* PCM fmt chunk size */
    hdr[20] = 1; hdr[21] = 0;              /* PCM format          */
    hdr[22] = chans & 0xff; hdr[23] = (chans >> 8) & 0xff;
    hdr[24] = sr & 0xff; hdr[25] = (sr >> 8) & 0xff;
    hdr[26] = (sr >> 16) & 0xff; hdr[27] = (sr >> 24) & 0xff;
    hdr[28] = byteRate & 0xff; hdr[29] = (byteRate >> 8) & 0xff;
    hdr[30] = (byteRate >> 16) & 0xff; hdr[31] = (byteRate >> 24) & 0xff;
    hdr[32] = block & 0xff; hdr[33] = (block >> 8) & 0xff;
    hdr[34] = bits & 0xff; hdr[35] = (bits >> 8) & 0xff;
    memcpy(hdr + 36, "data", 4);
    hdr[40] = dataSize & 0xff; hdr[41] = (dataSize >> 8) & 0xff;
    hdr[42] = (dataSize >> 16) & 0xff; hdr[43] = (dataSize >> 24) & 0xff;

    fwrite(hdr, 1, sizeof(hdr), stdout);
    if (ch->len)
        fwrite(ch->buf, 1, ch->len, stdout);
}

/* ------------------- audio destination callbacks ------------------- */

static ttsResultType GetVersion(char *strversion)
{
    strcpy(strversion, "1.0.0");
    return tts_OK;
}

static ttsResultType Open(void **channel, const char *DeviceName,
                          unsigned int SampleRate, ttsAudioCodingType coding,
                          ttsAudioSampleType nChannels, ttsBoolType *bRealTime,
                          const void *pUser)
{
    ChannelType *ch;
    (void)DeviceName;
    (void)pUser;

    if (bRealTime)
        *bRealTime = ttsFALSE; /* non-realtime, like an audio file */

    ch = (ChannelType *)calloc(1, sizeof(ChannelType));
    if (ch == NULL)
        return OUTOFMEMORY;

    ch->SampleRate = SampleRate;
    ch->coding = coding;
    ch->nChannels = nChannels;

    *channel = ch;

#if defined(_WIN32)
    /* Ensure stdout is in binary mode so PCM/WAV bytes are not mangled. */
    _setmode(_fileno(stdout), _O_BINARY);
#endif
    return tts_OK;
}

static ttsResultType Close(void *channel)
{
    ChannelType *ch = (ChannelType *)channel;
    if (ch == NULL)
        return tts_OK;
    free(ch->buf);
    free(ch);
    return tts_OK;
}

static ttsResultType PutData(void *channel, void *pData,
                             unsigned int nBytes, unsigned int *nSent)
{
    ChannelType *ch = (ChannelType *)channel;
    if (ch == NULL)
        return tts_OK;
    if (nSent)
        *nSent = 0;

    if (nBytes == 0)
    {
        /* End of stream: emit the complete WAV and flush. */
        emit_wav(ch);
        fflush(stdout);
        return tts_OK;
    }

    if (ensure_cap(ch, nBytes) != 0)
        return OUTOFMEMORY;
    memcpy(ch->buf + ch->len, pData, nBytes);
    ch->len += nBytes;
    return tts_OK;
}

static ttsResultType Stop(void *channel)
{
    (void)channel;
    return tts_OK;
}

static ttsResultType Pause(void *channel)
{
    (void)channel;
    return tts_OK;
}

static ttsResultType Resume(void *channel)
{
    (void)channel;
    return tts_OK;
}

static ttsResultType GetErrorMessage(ttsResultType ErrorCode,
                                     char *message, unsigned int size)
{
    const char *pmes;
    if (message == NULL || size == 0)
        return INVALIDPARAM;
    switch (ErrorCode)
    {
    case OUTOFMEMORY:  pmes = "Out of memory"; break;
    case INVALIDPARAM: pmes = "Invalid parameter"; break;
    default:           pmes = "Unknown error"; break;
    }
    strncpy(message, pmes, size - 1);
    message[size - 1] = '\0';
    return tts_OK;
}

/* ------------------- module entry point ------------------- */

ttspFuncType LoqTTS7AudioUser(ttsProcIdType procname, const void *pUser)
{
    static const struct {
        ttsProcIdType name;
        ttspFuncType func;
    } FuncTable[] = {
        {tts_GET_VERSION, (ttspFuncType)GetVersion},
        {TTSAUDIO_OPEN, (ttspFuncType)Open},
        {TTSAUDIO_CLOSE, (ttspFuncType)Close},
        {TTSAUDIO_PUTDATA, (ttspFuncType)PutData},
        {TTSAUDIO_STOP, (ttspFuncType)Stop},
        {TTSAUDIO_PAUSE, (ttspFuncType)Pause},
        {TTSAUDIO_RESUME, (ttspFuncType)Resume},
        {TTSAUDIO_GETERRORMESSAGE, (ttspFuncType)GetErrorMessage},
    };
    unsigned int i, size = sizeof(FuncTable) / sizeof(FuncTable[0]);
    (void)pUser;
    if (procname == NULL)
        return NULL;
    for (i = 0; i < size; i++)
    {
        if (strcmp(procname, FuncTable[i].name) == 0)
            return FuncTable[i].func;
    }
    return NULL;
}
