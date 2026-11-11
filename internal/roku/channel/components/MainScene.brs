sub init()
    m.top.setFocus(true)

    ' Screen nodes
    m.dashboard = m.top.findNode("dashboard")
    m.statusLabel = m.top.findNode("statusLabel")
    m.video = m.top.findNode("video")

    m.subtitleGroup = m.top.findNode("subtitleGroup")
    m.subtitleLabel = m.top.findNode("subtitleLabel")
    m.subtitleShadow = m.top.findNode("subtitleShadow")
    m.subtitleTimer = m.top.findNode("subtitleTimer")

    m.seekToast = m.top.findNode("seekToast")
    m.seekToastLabel = m.top.findNode("seekToastLabel")
    m.seekToastTimer = m.top.findNode("seekToastTimer")

    if m.video.seekMode <> invalid
        m.video.seekMode = "accurate"
    end if

    ' Main Player Deck nodes
    m.deckGroup = m.top.findNode("deckGroup")
    m.deckTitle = m.top.findNode("deckTitle")
    m.deckBadge = m.top.findNode("deckBadge")
    m.deckTime = m.top.findNode("deckTime")
    m.progressBarFill = m.top.findNode("progressBarFill")
    m.progressDot = m.top.findNode("progressDot")

    m.card0Bg = m.top.findNode("card0Bg")
    m.card0Label = m.top.findNode("card0Label")
    m.card1Bg = m.top.findNode("card1Bg")
    m.card1Label = m.top.findNode("card1Label")
    m.card2Bg = m.top.findNode("card2Bg")
    m.card2Label = m.top.findNode("card2Label")

    ' Subtitle & Audio Side Drawer nodes
    m.subPanelGroup = m.top.findNode("subPanelGroup")
    m.subRow0Bg = m.top.findNode("subRow0Bg")
    m.subRow0Val = m.top.findNode("subRow0Val")
    m.subRow1Bg = m.top.findNode("subRow1Bg")
    m.subRow1Val = m.top.findNode("subRow1Val")
    m.subRow2Bg = m.top.findNode("subRow2Bg")
    m.subRow2Val = m.top.findNode("subRow2Val")
    m.subRow3Bg = m.top.findNode("subRow3Bg")
    m.subRow3Val = m.top.findNode("subRow3Val")
    m.subRow4Bg = m.top.findNode("subRow4Bg")
    m.subRow4Val = m.top.findNode("subRow4Val")
    m.subRow5Bg = m.top.findNode("subRow5Bg")
    m.subRow5Val = m.top.findNode("subRow5Val")

    ' State machine constants
    m.STATE_PLAYING = 0
    m.STATE_DECK = 1
    m.STATE_SUB_PANEL = 2
    m.currentState = m.STATE_PLAYING

    ' Navigation indices
    m.deckCardIndex = 0 ' 0=Resume, 1=Subtitles, 2=Next Video
    m.subRowIndex = 0   ' 0..5

    ' Subtitle & audio state
    m.cues = []
    m.subOffset = 0.0
    m.subsVisible = true
    m.subSizeIndex = 1  ' 0=Normal (30pt), 1=Large (42pt - default), 2=Huge (54pt)
    m.subColorIndex = 0 ' 0=White, 1=Yellow, 2=Sky Blue
    m.colors = ["0xFFFFFFFF", "0xFBBF24FF", "0x38BDF8FF"]
    m.colorNames = ["White", "Yellow", "Sky Blue"]
    m.sizeNames = ["Normal (30pt)", "Large (42pt)", "Huge (54pt)"]
    m.subTracks = ["English [CC]", "OFF"]
    m.subTrackIndex = 0
    m.audioTracks = ["Track 1: Default"]
    m.audioTrackIndex = 0
    m.isShowingSample = false
    m.callbackUrl = ""
    m.currentTitle = "NOW PLAYING"

    ' Observers
    m.video.observeField("state", "onVideoStateChange")
    m.video.observeField("position", "onPositionChange")
    m.seekToastTimer.observeField("fire", "hideSeekToast")
    m.subtitleTimer.observeField("fire", "updateSubtitleDisplay")

    applySubtitleStyle()
end sub

sub onArgs()
    args = m.top.args
    if args = invalid then return

    mediaUrl = ""
    subUrl = ""
    title = ""
    offsetStr = ""

    if args.DoesExist("url") and args["url"] <> invalid then mediaUrl = args["url"]
    if args.DoesExist("media_url") and args["media_url"] <> invalid then mediaUrl = args["media_url"]
    if args.DoesExist("subtitles") and args["subtitles"] <> invalid then subUrl = args["subtitles"]
    if args.DoesExist("sub_url") and args["sub_url"] <> invalid then subUrl = args["sub_url"]
    if args.DoesExist("title") and args["title"] <> invalid then title = args["title"]
    if args.DoesExist("sub_offset") and args["sub_offset"] <> invalid then offsetStr = args["sub_offset"]
    if args.DoesExist("callback_url") and args["callback_url"] <> invalid then m.callbackUrl = args["callback_url"]

    if mediaUrl = "" or mediaUrl = "undefined"
        m.statusLabel.text = "Cannot Play: Invalid or Non-Video URL"
        m.statusLabel.color = "0xEF4444FF"
        m.dashboard.visible = true
        return
    end if

    if title <> ""
        m.currentTitle = title
    else
        m.currentTitle = "NOW PLAYING"
    end if
    m.deckTitle.text = UCase(m.currentTitle.ToStr())

    if offsetStr <> ""
        m.subOffset = Val(offsetStr)
    else
        m.subOffset = 0.0
    end if

    ' Reset UI state
    m.cues = []
    m.subsVisible = true
    m.subTrackIndex = 0
    m.isShowingSample = false
    m.subtitleGroup.visible = false
    m.subtitleGroup.translation = [0, 0]
    m.deckGroup.visible = false
    m.subPanelGroup.visible = false
    m.currentState = m.STATE_PLAYING

    m.dashboard.visible = false
    m.video.visible = true

    content = CreateObject("roSGNode", "ContentNode")
    content.url = mediaUrl
    content.title = m.currentTitle

    lUrl = LCase(mediaUrl)
    if InStr(1, lUrl, ".m3u8") > 0
        content.streamFormat = "hls"
        content.live = false
    else if InStr(1, lUrl, ".mpd") > 0
        content.streamFormat = "dash"
        content.live = false
    else
        content.streamFormat = "mp4"
    end if

    m.video.content = content
    m.video.control = "play"
    m.subtitleTimer.control = "start"

    ' Fetch subtitles if provided
    if subUrl <> ""
        m.subTracks = ["English [CC]", "OFF"]
        m.subTask = CreateObject("roSGNode", "SubtitleTask")
        m.subTask.observeField("cues", "onSubtitlesLoaded")
        m.subTask.url = subUrl
        m.subTask.control = "RUN"
    else
        m.subTracks = ["OFF"]
        m.subTrackIndex = 0
        m.subsVisible = false
    end if

    applySubtitleStyle()
    showToast("Starting Playback")
end sub

sub onSubtitlesLoaded(event as Object)
    m.cues = event.getData()
    if m.cues <> invalid and m.cues.Count() > 0
        if m.subsVisible then m.subtitleGroup.visible = true
        print "MainScene: Subtitles loaded: " + m.cues.Count().ToStr() + " cues"
    end if
end sub

sub updateSubtitleDisplay()
    if not m.subsVisible or m.cues = invalid or m.cues.Count() = 0
        if not m.isShowingSample
            m.subtitleLabel.text = ""
            m.subtitleShadow.text = ""
        end if
        return
    end if

    vidPos = m.video.position + m.subOffset
    currentText = ""

    for each cue in m.cues
        if vidPos >= cue.start and vidPos <= cue.endPos
            currentText = cue.text
            exit for
        else if vidPos < cue.start
            exit for
        end if
    end for

    if currentText <> ""
        if m.subtitleLabel.text <> currentText
            m.subtitleLabel.text = currentText
            m.subtitleShadow.text = currentText
        end if
        m.isShowingSample = false
        m.subtitleGroup.visible = true
    else
        if not m.isShowingSample and m.subtitleLabel.text <> ""
            m.subtitleLabel.text = ""
            m.subtitleShadow.text = ""
            m.subtitleGroup.visible = false
        end if
    end if
end sub

sub onPositionChange(event as Object)
    vidPos = event.getData()
    dur = m.video.duration

    if dur > 0
        m.deckTime.text = formatTime(vidPos) + " / " + formatTime(dur)
        ratio = vidPos / dur
        if ratio > 1.0 then ratio = 1.0
        fillWidth = Int(ratio * 1760)
        m.progressBarFill.width = fillWidth
        m.progressDot.translation = [80 + fillWidth - 8, 851]
    else
        m.deckTime.text = formatTime(vidPos)
        m.progressBarFill.width = 0
        m.progressDot.translation = [72, 851]
    end if
end sub

function formatTime(totalSeconds as Float) as String
    t = Int(totalSeconds)
    hours = Int(t / 3600)
    mins = Int((t Mod 3600) / 60)
    secs = t Mod 60

    if hours > 0
        return padZero(hours) + ":" + padZero(mins) + ":" + padZero(secs)
    else
        return padZero(mins) + ":" + padZero(secs)
    end if
end function

function padZero(n as Integer) as String
    if n < 10 then return "0" + n.ToStr()
    return n.ToStr()
end function

function formatOffset(offset as Float) as String
    if offset > 0.0
        return "+" + Str(offset).Trim() + "s"
    else if offset < 0.0
        return Str(offset).Trim() + "s"
    else
        return "0.0s (In Sync)"
    end if
end function

sub showToast(msg as String)
    if m.seekToast = invalid then return
    m.seekToastLabel.text = msg
    m.seekToast.visible = true
    m.seekToastTimer.control = "stop"
    m.seekToastTimer.control = "start"
end sub

sub hideSeekToast()
    if m.seekToast <> invalid then m.seekToast.visible = false
end sub

sub showDeck()
    if m.video = invalid then return
    m.currentState = m.STATE_DECK
    m.video.control = "pause"

    ' Shift subtitles upward slightly while deck is open so subtitles don't overlap
    m.subtitleGroup.translation = [0, -180]

    m.deckCardIndex = 0
    updateDeckCards()

    m.deckGroup.visible = true
    m.subPanelGroup.visible = false
    m.top.setFocus(true)
end sub

sub resumePlayback()
    if m.video = invalid then return
    m.currentState = m.STATE_PLAYING
    m.deckGroup.visible = false
    m.subPanelGroup.visible = false
    m.subtitleGroup.translation = [0, 0]

    if m.isShowingSample
        clearSubPreview()
    end if

    m.video.control = "resume"
    showToast("> Playing")
    m.top.setFocus(true)
end sub

sub openSubPanel()
    m.currentState = m.STATE_SUB_PANEL
    m.deckGroup.visible = false
    m.subPanelGroup.visible = true
    m.subRowIndex = 0

    ' Refresh audio tracks from video if available
    if m.video <> invalid and m.video.audioTracks <> invalid and m.video.audioTracks.Count() > 0
        tracks = []
        for i = 0 to m.video.audioTracks.Count() - 1
            t = m.video.audioTracks[i]
            desc = "Track " + (i + 1).ToStr()
            if t.DoesExist("Track") and t.Track <> ""
                desc = t.Track
            end if
            tracks.Push(desc)
        end for
        m.audioTracks = tracks
    end if

    updateSubPanelRows()
    updateSubPanelFocus()
    showSubPreview()
    m.top.setFocus(true)
end sub

sub closeSubPanel()
    clearSubPreview()
    m.subPanelGroup.visible = false
    m.currentState = m.STATE_DECK
    m.deckGroup.visible = true
    updateDeckCards()
    m.top.setFocus(true)
end sub

sub stopPlayback()
    if m.video <> invalid
        m.video.control = "stop"
        m.video.visible = false
    end if
    m.deckGroup.visible = false
    m.subPanelGroup.visible = false
    m.subtitleGroup.visible = false
    m.subtitleGroup.translation = [0, 0]
    m.seekToast.visible = false
    m.subtitleTimer.control = "stop"
    m.cues = []
    m.isShowingSample = false
    m.dashboard.visible = true
    m.top.setFocus(true)
end sub

sub updateDeckCards()
    if m.deckCardIndex = 0
        m.card0Bg.color = "0x2563EBEE"
        m.card0Label.color = "0xFFFFFFFF"
        m.card1Bg.color = "0x1E293BEE"
        m.card1Label.color = "0x94A3B8FF"
        m.card2Bg.color = "0x1E293BEE"
        m.card2Label.color = "0x94A3B8FF"
    else if m.deckCardIndex = 1
        m.card0Bg.color = "0x1E293BEE"
        m.card0Label.color = "0x94A3B8FF"
        m.card1Bg.color = "0x2563EBEE"
        m.card1Label.color = "0xFFFFFFFF"
        m.card2Bg.color = "0x1E293BEE"
        m.card2Label.color = "0x94A3B8FF"
    else
        m.card0Bg.color = "0x1E293BEE"
        m.card0Label.color = "0x94A3B8FF"
        m.card1Bg.color = "0x1E293BEE"
        m.card1Label.color = "0x94A3B8FF"
        m.card2Bg.color = "0x2563EBEE"
        m.card2Label.color = "0xFFFFFFFF"
    end if
end sub

sub updateSubPanelFocus()
    bgs = [m.subRow0Bg, m.subRow1Bg, m.subRow2Bg, m.subRow3Bg, m.subRow4Bg, m.subRow5Bg]
    for i = 0 to 5
        if i = m.subRowIndex
            bgs[i].color = "0x2563EBEE"
        else
            bgs[i].color = "0x0F172ACC"
        end if
    end for
end sub

sub updateSubPanelRows()
    if m.subTracks.Count() > 0 and m.subTrackIndex < m.subTracks.Count()
        m.subRow0Val.text = "< " + m.subTracks[m.subTrackIndex] + " >"
    else
        m.subRow0Val.text = "< OFF >"
    end if

    m.subRow1Val.text = "< " + m.sizeNames[m.subSizeIndex] + " >"
    m.subRow2Val.text = "< " + m.colorNames[m.subColorIndex] + " >"
    m.subRow3Val.text = "< " + formatOffset(m.subOffset) + " >"

    if m.audioTracks.Count() > 0 and m.audioTrackIndex < m.audioTracks.Count()
        m.subRow4Val.text = "< " + m.audioTracks[m.audioTrackIndex] + " >"
    else
        m.subRow4Val.text = "< Default >"
    end if
end sub

sub adjustSetting(rowIndex as Integer, delta as Integer)
    if rowIndex = 0
        if m.subTracks.Count() > 1
            m.subTrackIndex = (m.subTrackIndex + delta + m.subTracks.Count()) Mod m.subTracks.Count()
            trackName = m.subTracks[m.subTrackIndex]
            if trackName = "OFF"
                m.subsVisible = false
                m.subtitleGroup.visible = false
            else
                m.subsVisible = true
                showSubPreview()
            end if
        end if
    else if rowIndex = 1
        count = m.sizeNames.Count()
        m.subSizeIndex = (m.subSizeIndex + delta + count) Mod count
        applySubtitleStyle()
    else if rowIndex = 2
        count = m.colors.Count()
        m.subColorIndex = (m.subColorIndex + delta + count) Mod count
        applySubtitleStyle()
    else if rowIndex = 3
        if delta > 0
            m.subOffset = m.subOffset + 0.5
        else
            m.subOffset = m.subOffset - 0.5
        end if
    else if rowIndex = 4
        if m.audioTracks.Count() > 1
            m.audioTrackIndex = (m.audioTrackIndex + delta + m.audioTracks.Count()) Mod m.audioTracks.Count()
            if m.video <> invalid
                m.video.currentAudioTrack = m.audioTrackIndex
            end if
        end if
    end if

    updateSubPanelRows()
end sub

sub applySubtitleStyle()
    if m.subtitleLabel = invalid or m.subtitleShadow = invalid then return

    m.subtitleLabel.color = m.colors[m.subColorIndex]

    if m.subSizeIndex = 0
        m.subtitleLabel.font = "font:MediumBoldSystemFont"
        m.subtitleShadow.font = "font:MediumBoldSystemFont"
        m.subtitleGroup.scale = [1.0, 1.0]
    else if m.subSizeIndex = 1
        m.subtitleLabel.font = "font:LargeBoldSystemFont"
        m.subtitleShadow.font = "font:LargeBoldSystemFont"
        m.subtitleGroup.scale = [1.0, 1.0]
    else
        m.subtitleLabel.font = "font:LargeBoldSystemFont"
        m.subtitleShadow.font = "font:LargeBoldSystemFont"
        m.subtitleGroup.scaleRotateCenter = [960, 930]
        m.subtitleGroup.scale = [1.3, 1.3]
    end if
end sub

sub showSubPreview()
    if not m.subsVisible then return

    if m.subtitleLabel.text = "" or not m.subtitleGroup.visible
        sampleText = "[ Subtitle Preview - Audio Sync " + formatOffset(m.subOffset) + " ]"
        m.subtitleLabel.text = sampleText
        m.subtitleShadow.text = sampleText
        m.isShowingSample = true
    end if

    applySubtitleStyle()
    m.subtitleGroup.visible = true
end sub

sub clearSubPreview()
    if m.isShowingSample
        m.subtitleLabel.text = ""
        m.subtitleShadow.text = ""
        m.subtitleGroup.visible = false
        m.isShowingSample = false
    end if
end sub

sub seekRelative(seconds as Float)
    if m.video = invalid or m.video.duration <= 0 then return
    newPos = m.video.position + seconds
    if newPos < 0 then newPos = 0
    if newPos > m.video.duration then newPos = m.video.duration - 1
    m.video.seek = newPos

    sign = "+"
    if seconds < 0 then sign = "-"
    showToast(">> " + sign + Abs(seconds).ToStr() + "s (" + formatTime(newPos) + ")")
end sub

sub onVideoStateChange(event as Object)
    state = event.getData()
    print "MainScene: video state = " + state

    if state = "finished"
        if m.callbackUrl <> ""
            notifyBackend(m.callbackUrl + "?event=finished")
        end if
    else if state = "error"
        errCode = m.video.errorCode
        errMsg = m.video.errorMsg
        print "MainScene: video error code="; errCode; " msg="; errMsg
        m.statusLabel.text = "Playback Error: " + errMsg.ToStr()
        m.statusLabel.color = "0xEF4444FF"
        m.dashboard.visible = true
        m.video.visible = false
        m.subtitleGroup.visible = false
        showToast("Error: " + errMsg.ToStr())
    end if
end sub

sub notifyBackend(url as String)
    http = CreateObject("roUrlTransfer")
    http.SetCertificatesFile("common:/certs/ca-bundle.crt")
    http.InitClientCertificates()
    http.SetUrl(url)
    http.AsyncGetToString()
end sub

' ==============================================================
' REMOTE CONTROL HANDLING
' ==============================================================
function onKeyEvent(key as String, press as Boolean) as Boolean
    if not press then return false
    lKey = LCase(key)

    if m.video <> invalid and m.video.visible = true

        ' STATE 0: VIDEO PLAYING (UI HIDDEN)
        if m.currentState = m.STATE_PLAYING
            if lKey = "right" or lKey = "fastforward" or lKey = "fwd"
                seekRelative(15)
                return true
            else if lKey = "left" or lKey = "rewind" or lKey = "rev"
                seekRelative(-15)
                return true
            else if lKey = "replay" or lKey = "instantreplay"
                seekRelative(-10)
                return true
            else if lKey = "ok" or lKey = "select" or lKey = "down" or lKey = "up"
                showDeck()
                return true
            else if lKey = "play" or lKey = "pause"
                if m.video.state = "paused"
                    resumePlayback()
                else
                    showDeck()
                end if
                return true
            else if lKey = "back"
                stopPlayback()
                return true
            end if

        ' STATE 1: MAIN PLAYER DECK (PAUSED)
        else if m.currentState = m.STATE_DECK
            if lKey = "left"
                m.deckCardIndex = (m.deckCardIndex - 1 + 3) Mod 3
                updateDeckCards()
                return true
            else if lKey = "right"
                m.deckCardIndex = (m.deckCardIndex + 1) Mod 3
                updateDeckCards()
                return true
            else if lKey = "play"
                resumePlayback()
                return true
            else if lKey = "ok" or lKey = "select"
                if m.deckCardIndex = 0
                    resumePlayback()
                else if m.deckCardIndex = 1
                    openSubPanel()
                else if m.deckCardIndex = 2
                    ' Next Video / Skip
                    if m.callbackUrl <> ""
                        notifyBackend(m.callbackUrl + "?event=next")
                    else
                        stopPlayback()
                    end if
                end if
                return true
            else if lKey = "back"
                resumePlayback()
                return true
            end if

        ' STATE 2: SUBTITLE & AUDIO SIDE DRAWER
        else if m.currentState = m.STATE_SUB_PANEL
            if lKey = "up"
                m.subRowIndex = (m.subRowIndex - 1 + 6) Mod 6
                updateSubPanelFocus()
                return true
            else if lKey = "down"
                m.subRowIndex = (m.subRowIndex + 1) Mod 6
                updateSubPanelFocus()
                return true
            else if lKey = "left"
                adjustSetting(m.subRowIndex, -1)
                return true
            else if lKey = "right"
                adjustSetting(m.subRowIndex, 1)
                return true
            else if lKey = "ok" or lKey = "select"
                if m.subRowIndex = 3
                    m.subOffset = 0.0
                    updateSubPanelRows()
                    if m.isShowingSample
                        sampleText = "[ Subtitle Preview - Audio Sync 0.0s ]"
                        m.subtitleLabel.text = sampleText
                        m.subtitleShadow.text = sampleText
                    end if
                else if m.subRowIndex = 5
                    closeSubPanel()
                else
                    adjustSetting(m.subRowIndex, 1)
                end if
                return true
            else if lKey = "back"
                closeSubPanel()
                return true
            end if
        end if

    end if

    return false
end function
