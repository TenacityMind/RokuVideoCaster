sub init()
    m.top.setFocus(true)

    ' Screen nodes
    m.dashboard = m.top.findNode("dashboard")
    m.statusLabel = m.top.findNode("statusLabel")
    m.webUrlLabel = m.top.findNode("webUrlLabel")
    m.video = m.top.findNode("video")

    m.subtitleGroup = m.top.findNode("subtitleGroup")
    m.subtitleLabel = m.top.findNode("subtitleLabel")
    m.subtitleShadow = m.top.findNode("subtitleShadow")
    m.subtitleTimer = m.top.findNode("subtitleTimer")

    m.seekToast = m.top.findNode("seekToast")
    m.seekToastLabel = m.top.findNode("seekToastLabel")
    m.seekToastTimer = m.top.findNode("seekToastTimer")

    m.deckAutoHideTimer = m.top.findNode("deckAutoHideTimer")

    if m.video.seekMode <> invalid
        m.video.seekMode = "accurate"
    end if

    ' Netflix Header & Deck nodes
    m.topHeaderGroup = m.top.findNode("topHeaderGroup")
    m.deckGroup = m.top.findNode("deckGroup")
    m.deckTitle = m.top.findNode("deckTitle")
    m.deckTimeElapsed = m.top.findNode("deckTimeElapsed")
    m.deckTimeRemaining = m.top.findNode("deckTimeRemaining")
    m.progressBarFill = m.top.findNode("progressBarFill")
    m.progressDot = m.top.findNode("progressDot")

    ' Netflix Floating Scene Preview Card
    m.scrubThumbGroup = m.top.findNode("scrubThumbGroup")
    m.scrubThumbPoster = m.top.findNode("scrubThumbPoster")
    m.scrubThumbTime = m.top.findNode("scrubThumbTime")

    ' 5 Uniform Action Buttons
    m.card0Bg = m.top.findNode("card0Bg")
    m.card0Label = m.top.findNode("card0Label")
    m.card1Bg = m.top.findNode("card1Bg")
    m.card1Label = m.top.findNode("card1Label")
    m.card2Bg = m.top.findNode("card2Bg")
    m.card2Label = m.top.findNode("card2Label")
    m.card3Bg = m.top.findNode("card3Bg")
    m.card3Label = m.top.findNode("card3Label")
    m.card4Bg = m.top.findNode("card4Bg")
    m.card4Label = m.top.findNode("card4Label")

    ' Subtitle & Audio Center Modal nodes
    m.subPanelGroup = m.top.findNode("subPanelGroup")
    m.audioRow0Bg = m.top.findNode("audioRow0Bg")
    m.audioRow0Val = m.top.findNode("audioRow0Val")
    m.audioRow1Bg = m.top.findNode("audioRow1Bg")
    m.audioRow1Val = m.top.findNode("audioRow1Val")

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

    ' State machine constants
    m.STATE_PLAYING = 0
    m.STATE_DECK = 1
    m.STATE_MODAL = 2
    m.STATE_SCRUBBING = 3
    m.currentState = m.STATE_PLAYING

    ' Navigation indices
    m.deckCardIndex = 0 ' 0=Play/Pause, 1=Rewind 10s, 2=Forward 10s, 3=Audio & Subs, 4=Next
    m.subRowIndex = 0   ' 0..4

    ' Scrubbing and thumbnail state
    m.scrubTargetPos = 0.0
    m.thumbBaseUrl = ""

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
    m.deckAutoHideTimer.observeField("fire", "onDeckAutoHide")

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
    if args.DoesExist("offset") and args["offset"] <> invalid then offsetStr = args["offset"].ToStr()
    if args.DoesExist("callback_url") and args["callback_url"] <> invalid then m.callbackUrl = args["callback_url"]

    ' If web_url passed on launch, update the dashboard URL pill
    if args.DoesExist("web_url") and args["web_url"] <> invalid and args["web_url"] <> ""
        m.webUrlLabel.text = args["web_url"]
    end if

    if mediaUrl = ""
        ' No media URL: stay on dashboard
        m.dashboard.visible = true
        m.video.visible = false
        m.deckGroup.visible = false
        m.subPanelGroup.visible = false
        m.topHeaderGroup.visible = false
        m.statusLabel.text = "* READY TO CAST"
        m.statusLabel.color = "0x34d399FF"
        return
    end if

    if title <> ""
        m.currentTitle = title
    else
        m.currentTitle = "NOW PLAYING"
    end if
    m.deckTitle.text = UCase(m.currentTitle.ToStr())

    ' Derive thumbnail base URL (check explicit thumb_url param or fall back to /media/ path)
    m.thumbBaseUrl = ""
    if args.DoesExist("thumb_url") and args.thumb_url <> ""
        m.thumbBaseUrl = args.thumb_url
    else if InStr(1, mediaUrl, "/media/") > 0
        parts = mediaUrl.Split("/")
        if parts.Count() >= 5
            m.thumbBaseUrl = parts[0] + "//" + parts[2] + "/media/" + parts[4] + "/thumb"
        end if
    end if

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
    m.topHeaderGroup.visible = false
    m.deckGroup.visible = false
    m.subPanelGroup.visible = false
    m.scrubThumbGroup.visible = false
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
    else if InStr(1, lUrl, ".mkv") > 0
        content.streamFormat = "mkv"
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
    if m.currentState = m.STATE_SCRUBBING then return

    vidPos = event.getData()
    dur = m.video.duration

    if dur > 0
        m.deckTimeElapsed.text = formatTime(vidPos)
        remSec = dur - vidPos
        if remSec < 0 then remSec = 0
        m.deckTimeRemaining.text = "-" + formatTime(remSec)

        ratio = vidPos / dur
        if ratio > 1.0 then ratio = 1.0
        fillWidth = Int(ratio * 1760)
        m.progressBarFill.width = fillWidth
        m.progressDot.translation = [80 + fillWidth - 8, 840]
    else
        m.deckTimeElapsed.text = formatTime(vidPos)
        m.deckTimeRemaining.text = "-00:00:00"
        m.progressBarFill.width = 0
        m.progressDot.translation = [72, 840]
    end if
end sub

function formatTime(totalSeconds as Float) as String
    t = Int(totalSeconds)
    if t < 0 then t = 0
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

' ==============================================================
' NETFLIX CONTROLS OVERLAY (DECK)
' ==============================================================
sub pauseAndShowDeck()
    if m.video = invalid then return
    m.video.control = "pause"
    m.currentState = m.STATE_DECK
    m.card0Label.text = "> Play"

    m.subtitleGroup.translation = [0, -180]
    m.deckCardIndex = 0
    updateDeckCards()

    m.topHeaderGroup.visible = true
    m.deckGroup.visible = true
    m.scrubThumbGroup.visible = false
    m.subPanelGroup.visible = false
    m.deckAutoHideTimer.control = "stop"
    m.top.setFocus(true)
end sub

sub showDeckNoPause()
    if m.video = invalid then return
    m.currentState = m.STATE_DECK
    m.card0Label.text = "|| Pause"

    m.subtitleGroup.translation = [0, -180]
    m.deckCardIndex = 0
    updateDeckCards()

    m.topHeaderGroup.visible = true
    m.deckGroup.visible = true
    m.scrubThumbGroup.visible = false
    m.subPanelGroup.visible = false

    ' If video is playing, start 5s auto-hide timer
    m.deckAutoHideTimer.control = "stop"
    if m.video.state = "playing"
        m.deckAutoHideTimer.control = "start"
    end if

    m.top.setFocus(true)
end sub

sub hideDeck()
    m.currentState = m.STATE_PLAYING
    m.deckGroup.visible = false
    m.topHeaderGroup.visible = false
    m.scrubThumbGroup.visible = false
    m.subPanelGroup.visible = false
    m.subtitleGroup.translation = [0, 0]
    m.deckAutoHideTimer.control = "stop"
    m.top.setFocus(true)
end sub

sub onDeckAutoHide()
    if m.currentState = m.STATE_DECK and m.video <> invalid and m.video.state = "playing"
        hideDeck()
    end if
end sub

sub togglePlayback()
    if m.video = invalid then return
    if m.video.state = "playing"
        m.video.control = "pause"
        m.card0Label.text = "> Play"
        showToast("Paused")
        m.deckAutoHideTimer.control = "stop"
    else
        m.video.control = "resume"
        m.card0Label.text = "|| Pause"
        showToast("Playing")
        m.deckAutoHideTimer.control = "start"
    end if
    updateDeckCards()
end sub

sub updateDeckCards()
    bgs = [m.card0Bg, m.card1Bg, m.card2Bg, m.card3Bg, m.card4Bg]
    lbls = [m.card0Label, m.card1Label, m.card2Label, m.card3Label, m.card4Label]

    for i = 0 to 4
        if i = m.deckCardIndex
            bgs[i].color = "0xE50914FF" ' Netflix Red focus
            lbls[i].color = "0xFFFFFFFF"
        else
            bgs[i].color = "0x1E1E24CC" ' Dark idle button
            lbls[i].color = "0xCCCCCCFF"
        end if
    end for
end sub

sub executeDeckAction(idx as Integer)
    if idx = 0
        togglePlayback()
    else if idx = 1
        seekRelative(-10)
    else if idx = 2
        seekRelative(10)
    else if idx = 3
        openSubPanel()
    else if idx = 4
        ' Next Video
        if m.callbackUrl <> ""
            notifyBackend(m.callbackUrl + "?event=next")
        else
            stopPlayback()
        end if
    end if
end sub

' ==============================================================
' SCENE SLIDER PREVIEW (THUMBNAIL SCRUBBING)
' ==============================================================
sub startScrubbing(delta as Float)
    if m.video = invalid or m.video.duration <= 0 then return
    m.currentState = m.STATE_SCRUBBING
    m.scrubTargetPos = m.video.position + delta
    clampAndRenderScrub()

    m.subtitleGroup.translation = [0, -180]
    m.deckGroup.visible = true
    m.topHeaderGroup.visible = true
    m.scrubThumbGroup.visible = true
    m.subPanelGroup.visible = false
    m.deckAutoHideTimer.control = "stop"
    m.top.setFocus(true)
end sub

sub adjustScrub(delta as Float)
    m.scrubTargetPos = m.scrubTargetPos + delta
    clampAndRenderScrub()
end sub

sub clampAndRenderScrub()
    dur = m.video.duration
    if dur <= 0 then return

    if m.scrubTargetPos < 0 then m.scrubTargetPos = 0
    if m.scrubTargetPos > dur then m.scrubTargetPos = dur

    ratio = m.scrubTargetPos / dur
    if ratio > 1.0 then ratio = 1.0
    fillWidth = Int(ratio * 1760)

    m.progressBarFill.width = fillWidth
    m.progressDot.translation = [80 + fillWidth - 8, 840]

    m.deckTimeElapsed.text = formatTime(m.scrubTargetPos)
    remSec = dur - m.scrubTargetPos
    if remSec < 0 then remSec = 0
    m.deckTimeRemaining.text = "-" + formatTime(remSec)

    ' Position floating scene thumbnail preview card (280px wide, raised to 595px to float cleanly above timeline)
    thumbX = 80 + fillWidth - 140
    if thumbX < 40 then thumbX = 40
    if thumbX > 1600 then thumbX = 1600
    m.scrubThumbGroup.translation = [thumbX, 595]

    m.scrubThumbTime.text = formatTime(m.scrubTargetPos)

    ' Update thumbnail poster if available
    if m.thumbBaseUrl <> ""
        m.scrubThumbPoster.uri = m.thumbBaseUrl + "?t=" + Int(m.scrubTargetPos).ToStr()
    else
        m.scrubThumbPoster.uri = ""
    end if
end sub

sub commitScrubAndPlay()
    if m.video = invalid then return
    m.video.seek = m.scrubTargetPos
    m.scrubThumbGroup.visible = false
    hideDeck()

    if m.video.state <> "playing"
        m.video.control = "resume"
    end if

    showToast("Playing from " + formatTime(m.scrubTargetPos))
end sub

sub cancelScrub()
    m.scrubThumbGroup.visible = false
    hideDeck()
    showToast("Cancelled")
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

' ==============================================================
' SUBTITLE & AUDIO CENTER MODAL
' ==============================================================
sub openSubPanel()
    m.currentState = m.STATE_MODAL
    m.deckAutoHideTimer.control = "stop"
    m.deckGroup.visible = false
    m.topHeaderGroup.visible = false
    m.scrubThumbGroup.visible = false
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
    m.subtitleGroup.translation = [0, 0]
    hideDeck()
end sub

sub updateSubPanelFocus()
    subBgs = [m.subRow0Bg, m.subRow1Bg, m.subRow2Bg, m.subRow3Bg, m.subRow4Bg]
    for i = 0 to 4
        if i = m.subRowIndex
            subBgs[i].color = "0xE50914FF" ' Netflix Red focus
        else
            subBgs[i].color = "0x15161DCC" ' Dark idle row
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
        m.audioRow0Val.text = m.audioTracks[m.audioTrackIndex] + " *"
    else
        m.audioRow0Val.text = "Track 1: Default *"
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

sub stopPlayback()
    if m.video <> invalid
        m.video.control = "stop"
        m.video.visible = false
    end if
    m.topHeaderGroup.visible = false
    m.deckGroup.visible = false
    m.subPanelGroup.visible = false
    m.scrubThumbGroup.visible = false
    m.subtitleGroup.visible = false
    m.subtitleGroup.translation = [0, 0]
    m.seekToast.visible = false
    m.subtitleTimer.control = "stop"
    m.deckAutoHideTimer.control = "stop"
    m.cues = []
    m.isShowingSample = false
    m.statusLabel.text = "* READY TO CAST"
    m.statusLabel.color = "0x34d399FF"
    m.dashboard.visible = true
    m.top.setFocus(true)
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
    if url = invalid or url = "" then return
    task = CreateObject("roSGNode", "NotifyTask")
    task.url = url
    task.control = "RUN"
end sub

' ==============================================================
' REMOTE CONTROL HANDLING (NETFLIX STANDARD)
' ==============================================================
function onKeyEvent(key as String, press as Boolean) as Boolean
    if not press then return false
    lKey = LCase(key)

    if m.video <> invalid and m.video.visible = true

        ' STATE 0: VIDEO PLAYING (CLEAN VIEW)
        if m.currentState = m.STATE_PLAYING
            if lKey = "left" or lKey = "rewind" or lKey = "rev"
                ' Left: Start Scene Slider Preview backward
                startScrubbing(-15)
                return true
            else if lKey = "right" or lKey = "fastforward" or lKey = "fwd"
                ' Right: Start Scene Slider Preview forward
                startScrubbing(15)
                return true
            else if lKey = "replay" or lKey = "instantreplay"
                startScrubbing(-10)
                return true
            else if lKey = "ok" or lKey = "select"
                ' OK: Pause video and show options deck
                pauseAndShowDeck()
                return true
            else if lKey = "down"
                ' Down: Show options deck WITHOUT pausing video
                showDeckNoPause()
                return true
            else if lKey = "up"
                ' Up: Jump directly into Audio & Subtitles modal
                openSubPanel()
                return true
            else if lKey = "play" or lKey = "pause"
                togglePlayback()
                return true
            else if lKey = "back"
                stopPlayback()
                return true
            end if

        ' STATE 3: SCENE SLIDER PREVIEW (SCRUBBING)
        else if m.currentState = m.STATE_SCRUBBING
            if lKey = "left" or lKey = "rewind" or lKey = "rev"
                adjustScrub(-15)
                return true
            else if lKey = "right" or lKey = "fastforward" or lKey = "fwd"
                adjustScrub(15)
                return true
            else if lKey = "ok" or lKey = "select" or lKey = "play"
                ' OK commits seek and starts playback from here!
                commitScrubAndPlay()
                return true
            else if lKey = "back"
                ' Back cancels scrubbing, returns to playing at original position
                cancelScrub()
                return true
            end if

        ' STATE 1: CONTROLS DECK OVERLAY
        else if m.currentState = m.STATE_DECK
            m.deckAutoHideTimer.control = "stop"
            if m.video.state = "playing" then m.deckAutoHideTimer.control = "start"

            if lKey = "left"
                m.deckCardIndex = (m.deckCardIndex - 1 + 5) Mod 5
                updateDeckCards()
                return true
            else if lKey = "right"
                m.deckCardIndex = (m.deckCardIndex + 1) Mod 5
                updateDeckCards()
                return true
            else if lKey = "play" or lKey = "pause"
                togglePlayback()
                return true
            else if lKey = "up"
                openSubPanel()
                return true
            else if lKey = "down"
                hideDeck()
                return true
            else if lKey = "ok" or lKey = "select"
                executeDeckAction(m.deckCardIndex)
                return true
            else if lKey = "back"
                hideDeck()
                return true
            end if

        ' STATE 2: SUBTITLE & AUDIO CENTER MODAL
        else if m.currentState = m.STATE_MODAL
            if lKey = "up"
                m.subRowIndex = (m.subRowIndex - 1 + 5) Mod 5
                updateSubPanelFocus()
                return true
            else if lKey = "down"
                m.subRowIndex = (m.subRowIndex + 1) Mod 5
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
                else if m.subRowIndex = 4
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
