sub init()
    m.top.functionName = "fetchAndParseSubtitles"
end sub

sub fetchAndParseSubtitles()
    url = m.top.url
    if url = invalid or url = "" then return
    print "SubtitleTask: fetching " + url

    transfer = CreateObject("roUrlTransfer")
    transfer.SetCertificatesFile("common:/certs/ca-bundle.crt")
    transfer.InitClientCertificates()
    transfer.SetUrl(url)
    transfer.EnableEncodings(true)
    transfer.AddHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
    raw = transfer.GetToString()
    if raw = invalid or Len(raw) = 0
        print "SubtitleTask: empty response or fetch failure"
        return
    end if
    print "SubtitleTask: received " + Len(raw).ToStr() + " bytes"

    cues = parseSubtitleContent(raw)
    print "SubtitleTask: parsed " + cues.Count().ToStr() + " cues"
    m.top.cues = cues
end sub

function parseSubtitleContent(raw as String) as Object
    cues = []
    if raw = invalid or Len(raw) = 0 then return cues

    ' Normalize CRLF to LF
    raw = raw.Replace(Chr(13), "")
    lines = raw.Split(Chr(10))

    curStart = -1.0
    curEnd = -1.0
    curText = ""

    for each line in lines
        trimmed = line.Trim()
        if InStr(1, trimmed, "-->") > 0
            ' If we had an in-progress cue, commit it
            if curStart >= 0.0 and curEnd > curStart and curText <> ""
                cues.Push({ start: curStart, endPos: curEnd, text: curText.Trim() })
            end if

            curText = ""
            times = trimmed.Split("-->")
            if times.Count() >= 2
                curStart = parseTimestamp(times[0])
                ' The end time may have extra tokens like "line:85% align:middle"
                endPart = times[1].Trim()
                endTokens = endPart.Split(" ")
                curEnd = parseTimestamp(endTokens[0])
            else
                curStart = -1.0
                curEnd = -1.0
            end if
        else if trimmed = ""
            ' Blank line ends the current cue
            if curStart >= 0.0 and curEnd > curStart and curText <> ""
                cues.Push({ start: curStart, endPos: curEnd, text: curText.Trim() })
                curStart = -1.0
                curEnd = -1.0
                curText = ""
            end if
        else
            ' Check if it's WEBVTT header or note
            if Left(trimmed, 6) = "WEBVTT" or Left(trimmed, 4) = "NOTE"
                ' Ignore header/comment
            else if curStart >= 0.0
                ' This is cue text line. Strip HTML tags like <i>, <b>, <font...>
                cleanLine = stripTags(trimmed)
                if cleanLine <> ""
                    if curText = ""
                        curText = cleanLine
                    else
                        curText = curText + Chr(10) + cleanLine
                    end if
                end if
            end if
        end if
    end for

    ' Flush trailing cue if file didn't end with blank line
    if curStart >= 0.0 and curEnd > curStart and curText <> ""
        cues.Push({ start: curStart, endPos: curEnd, text: curText.Trim() })
    end if

    return cues
end function

function parseTimestamp(ts as String) as Float
    ts = ts.Trim()
    ' Replace comma with dot for SRT compatibility
    ts = ts.Replace(",", ".")
    parts = ts.Split(":")
    if parts.Count() = 3
        hours = Val(parts[0])
        mins = Val(parts[1])
        secs = Val(parts[2])
        return (hours * 3600.0) + (mins * 60.0) + secs
    else if parts.Count() = 2
        mins = Val(parts[0])
        secs = Val(parts[1])
        return (mins * 60.0) + secs
    end if
    return 0.0
end function

function stripTags(s as String) as String
    res = ""
    inTag = false
    for i = 1 to Len(s)
        ch = Mid(s, i, 1)
        if ch = "<"
            inTag = true
        else if ch = ">"
            inTag = false
        else if not inTag
            res = res + ch
        end if
    end for
    return res
end function
