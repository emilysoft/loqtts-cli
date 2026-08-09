/************************************************************************/
/*                                                                      */
/*                loqtts - command line interface for Loquendo TTS 7    */
/*                                                                      */
/************************************************************************/

#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include "loqtts.h"

static const char *g_prog = "loqtts";

static void usage(FILE *out)
{
    fprintf(out,
        "Usage: %s [options] [text]\n"
        "\n"
        "Speaks the given text with a Loquendo TTS 7 voice. If no [text] is\n"
        "given, text is read from stdin (or from a file with -f).\n"
        "\n"
        "Options:\n"
        "  -v, --voice <name>     voice to use (default: first installed voice)\n"
        "  -l, --language <name>  language to use (default: voice's mother tongue)\n"
        "  -s, --style <name>     style to use (default: none)\n"
        "  -r, --rate <wpm>       absolute speed in words per minute\n"
        "  -p, --pitch <hz>       absolute pitch in hertz\n"
        "  -a, --volume <0-100>   volume (default: 50)\n"
        "  -o, --output <file>    write speech to a file instead of the audio board.\n"
        "                         .wav extension -> WAV file, otherwise raw PCM.\n"
        "  -f, --file <path>      read the text to speak from a file\n"
        "      --rate-hz <n>      audio sample rate for file output (default: 32000)\n"
        "      --mono             mono output instead of stereo\n"
        "      --list-voices      list all installed voices and exit\n"
        "      --list-languages   list all installed languages and exit\n"
        "      --version          print Loquendo TTS version and exit\n"
        "  -h, --help             show this help and exit\n"
        "\n"
        "Examples:\n"
        "  %s \"Hello world\"\n"
        "  %s -v Carmen -r 180 -p 200 \"Hola, ¿cómo estás?\"\n"
        "  %s -v Carlos -o out.wav -f speech.txt\n"
        "  cat speech.txt | %s\n"
        "  %s --list-voices\n",
        g_prog, g_prog, g_prog, g_prog, g_prog, g_prog);
}

static void print_error(const char *where, ttsResultType r)
{
    fprintf(stderr, "%s: %s: %s\n", g_prog, where, ttsGetErrorMessage(r));
}

static void *xrealloc(void *ptr, size_t size)
{
    void *p = realloc(ptr, size);
    if (!p)
    {
        fprintf(stderr, "%s: out of memory\n", g_prog);
        exit(2);
    }
    return p;
}

static char *read_all(FILE *f)
{
    size_t cap = 65536;
    size_t len = 0;
    char *buf = (char *)malloc(cap);
    if (!buf)
    {
        fprintf(stderr, "%s: out of memory\n", g_prog);
        exit(2);
    }
    for (;;)
    {
        size_t n = fread(buf + len, 1, cap - len - 1, f);
        len += n;
        if (n == 0)
            break;
        if (len + 1 >= cap)
        {
            cap *= 2;
            buf = (char *)xrealloc(buf, cap);
        }
    }
    buf[len] = '\0';
    return buf;
}

static char *read_file(const char *path)
{
    FILE *f = fopen(path, "rb");
    char *buf;
    if (!f)
    {
        fprintf(stderr, "%s: cannot open '%s'\n", g_prog, path);
        exit(2);
    }
    buf = read_all(f);
    fclose(f);
    return buf;
}

static ttsResultType list_objects(ttsHandleType hSession, ttsQueryType obj,
                                  const char *title)
{
    ttsHandleType hEnum = NULL;
    char *id = NULL;
    ttsResultType r = ttsEnumFirst(&hEnum, hSession, obj, NULL, &id);
    if (r != tts_OK)
        return r;
    printf("%s:\n", title);
    if (id)
    {
        do
        {
            char desc[ttsSTRINGMAXLEN];
            printf("  %s", id);
            if (ttsQueryAttribute(hSession, obj, "Description", id,
                                  desc, ttsSTRINGMAXLEN) == tts_OK &&
                desc[0])
                printf(" - %s", desc);
            if (obj == TTSOBJECTVOICE)
            {
                char extra[ttsSTRINGMAXLEN];
                if (ttsQueryAttribute(hSession, obj, "Gender", id,
                                      extra, ttsSTRINGMAXLEN) == tts_OK &&
                    extra[0])
                    printf(" (%s", extra);
                if (ttsQueryAttribute(hSession, obj, "Age", id,
                                      extra, ttsSTRINGMAXLEN) == tts_OK &&
                    extra[0])
                    printf("/%s)", extra);
                else if (desc[0])
                    printf(")");
            }
            printf("\n");
        } while (ttsEnumNext(hEnum, &id) == tts_OK && id);
    }
    ttsEnumClose(hEnum);
    return tts_OK;
}

static ttsResultType pick_default_voice(ttsHandleType hSession, char *buf,
                                        size_t bufsize)
{
    ttsHandleType hEnum = NULL;
    char *id = NULL;
    ttsResultType r = ttsEnumFirst(&hEnum, hSession, TTSOBJECTVOICE, NULL, &id);
    if (r != tts_OK)
        return r;
    if (id)
        strncpy(buf, id, bufsize - 1);
    else
        buf[0] = '\0';
    ttsEnumClose(hEnum);
    return tts_OK;
}

int main(int argc, char **argv)
{
    const char *voice = NULL;
    const char *language = NULL;
    const char *style = NULL;
    const char *outfile = NULL;
    const char *infile = NULL;
    const char *text = NULL;
    int speed = -1;
    int pitch = -1;
    int volume = -1;
    int sample_rate = 32000;
    int mono = 0;
    int list_voices = 0;
    int list_languages = 0;
    int do_version = 0;
    int i;

    for (i = 1; i < argc; i++)
    {
        const char *arg = argv[i];
        if (!strcmp(arg, "-h") || !strcmp(arg, "--help"))
        {
            usage(stdout);
            return 0;
        }
        else if (!strcmp(arg, "--list-voices"))
            list_voices = 1;
        else if (!strcmp(arg, "--list-languages"))
            list_languages = 1;
        else if (!strcmp(arg, "--version"))
            do_version = 1;
        else if (!strcmp(arg, "-v") || !strcmp(arg, "--voice"))
        {
            if (++i >= argc) { usage(stderr); return 2; }
            voice = argv[i];
        }
        else if (!strcmp(arg, "-l") || !strcmp(arg, "--language"))
        {
            if (++i >= argc) { usage(stderr); return 2; }
            language = argv[i];
        }
        else if (!strcmp(arg, "-s") || !strcmp(arg, "--style"))
        {
            if (++i >= argc) { usage(stderr); return 2; }
            style = argv[i];
        }
        else if (!strcmp(arg, "-r") || !strcmp(arg, "--rate"))
        {
            if (++i >= argc) { usage(stderr); return 2; }
            speed = atoi(argv[i]);
        }
        else if (!strcmp(arg, "-p") || !strcmp(arg, "--pitch"))
        {
            if (++i >= argc) { usage(stderr); return 2; }
            pitch = atoi(argv[i]);
        }
        else if (!strcmp(arg, "-a") || !strcmp(arg, "--volume"))
        {
            if (++i >= argc) { usage(stderr); return 2; }
            volume = atoi(argv[i]);
        }
        else if (!strcmp(arg, "-o") || !strcmp(arg, "--output"))
        {
            if (++i >= argc) { usage(stderr); return 2; }
            outfile = argv[i];
        }
        else if (!strcmp(arg, "-f") || !strcmp(arg, "--file"))
        {
            if (++i >= argc) { usage(stderr); return 2; }
            infile = argv[i];
        }
        else if (!strcmp(arg, "--rate-hz"))
        {
            if (++i >= argc) { usage(stderr); return 2; }
            sample_rate = atoi(argv[i]);
        }
        else if (!strcmp(arg, "--mono"))
            mono = 1;
        else if (arg[0] == '-' && arg[1])
        {
            fprintf(stderr, "%s: unknown option '%s'\n", g_prog, arg);
            usage(stderr);
            return 2;
        }
        else
            text = arg;
    }

    if (list_voices && list_languages)
    {
        fprintf(stderr, "%s: use --list-voices or --list-languages, not both\n",
                g_prog);
        return 2;
    }

    ttsHandleType hSession = NULL;
    ttsHandleType hReader = NULL;
    ttsResultType r = ttsNewSession(&hSession, NULL);
    if (r != tts_OK)
    {
        print_error("cannot create session", r);
        ttsDeleteSession(NULL);
        return 1;
    }
    r = ttsNewReader(&hReader, hSession);
    if (r != tts_OK)
    {
        print_error("cannot create reader", r);
        ttsDeleteSession(hSession);
        return 1;
    }

    if (do_version)
    {
        ttsInfoStringType ver;
        if (ttsGetVersionInfo(ver) == tts_OK)
            printf("%s\n", ver);
        else
            printf("Loquendo TTS 7\n");
    }

    if (list_voices)
    {
        r = list_objects(hSession, TTSOBJECTVOICE, "Installed voices");
        if (r != tts_OK)
            print_error("cannot enumerate voices", r);
        ttsDeleteSession(hSession);
        return r == tts_OK ? 0 : 1;
    }

    if (list_languages)
    {
        r = list_objects(hSession, TTSOBJECTLANGUAGE, "Installed languages");
        if (r != tts_OK)
            print_error("cannot enumerate languages", r);
        ttsDeleteSession(hSession);
        return r == tts_OK ? 0 : 1;
    }

    if (do_version)
    {
        ttsDeleteSession(hSession);
        return 0;
    }

    char defvoice[ttsSTRINGMAXLEN];
    if (!voice)
    {
        r = pick_default_voice(hSession, defvoice, sizeof(defvoice));
        if (r != tts_OK || !defvoice[0])
        {
            fprintf(stderr, "%s: no voice available\n", g_prog);
            ttsDeleteSession(hSession);
            return 1;
        }
        voice = defvoice;
    }

    if (infile && text)
    {
        fprintf(stderr, "%s: use either -f or a text argument, not both\n", g_prog);
        ttsDeleteSession(hSession);
        return 2;
    }

    char *owned = NULL;
    if (infile)
        text = owned = read_file(infile);
    else if (!text)
    {
        owned = read_all(stdin);
        text = owned;
    }

    if (!text || !text[0])
    {
        fprintf(stderr, "%s: no text to speak\n", g_prog);
        free(owned);
        ttsDeleteSession(hSession);
        return 2;
    }

    fprintf(stderr, "Using voice: %s\n", voice);
    if (language)
        fprintf(stderr, "Using language: %s\n", language);

    r = ttsLoadPersona(hReader, voice, language, style);
    if (r != tts_OK)
    {
        print_error("cannot load persona", r);
        free(owned);
        ttsDeleteSession(hSession);
        return 1;
    }

    if (speed >= 0)
    {
        r = ttsSetAbsoluteSpeed(hReader, speed);
        if (r != tts_OK)
            print_error("cannot set speed", r);
    }
    if (pitch >= 0)
    {
        r = ttsSetAbsolutePitch(hReader, pitch);
        if (r != tts_OK)
            print_error("cannot set pitch", r);
    }
    if (volume >= 0)
    {
        r = ttsSetVolume(hReader, volume);
        if (r != tts_OK)
            print_error("cannot set volume", r);
    }

    if (outfile)
        r = ttsSetAudio(hReader, "LTTS7AudioFile", outfile, sample_rate,
                        tts_LINEAR, mono ? tts_MONO : tts_STEREO, 0);
    else
        r = ttsSetAudio(hReader, "LTTS7AudioBoard", NULL, sample_rate,
                        tts_LINEAR, tts_STEREO, 0);

    if (r != tts_OK)
    {
        print_error("cannot open audio destination", r);
        free(owned);
        ttsDeleteSession(hSession);
        return 1;
    }

    r = ttsRead(hReader, text, ttsFALSE, ttsFALSE, NULL);
    if (r != tts_OK)
        print_error("cannot read text", r);

    free(owned);
    ttsDeleteSession(hSession);
    return r == tts_OK ? 0 : 1;
}
