sub init()
    m.top.setFocus(true)

    ' Screen nodes
    m.dashboard = m.top.findNode("dashboard")
    m.video = m.top.findNode("video")
    m.subtitleGroup = m.top.findNode("subtitleGroup")
    m.subtitleLabel = m.top.findNode("subtitleLabel")
    m.subtitleShadow = m.top.findNode("subtitleShadow")
    m.subSyncOSD = m.top.findNode("subSyncOSD")
    m.subSyncLabel = m.top.findNode("subSyncLabel")
    m.subSyncTimer = m.top.findNode("subSyncTimer")
    m.seekToast = m.top.findNode("seekToast")
    m.seekToastLabel = m.top.findNode("seekToastLabel")
    m.seekToastTimer = m.top.findNode("seekToastTimer")
    m.subtitleTimer = m.top.findNode("subtitleTimer")

    ' Configure video seek accuracy
    if m.video.seekMode <> invalid
        m.video.seekMode = "accurate"
    end if

    ' OSD Nodes (Interactive Settings Deck)
    m.osd = m.top.findNode("osd")
    m.osdTitle = m.top.findNode("osdTitle")
    m.osdRemaining = m.top.findNode("osdRemaining")
    m.osdCurrentTime = m.top.findNode("osdCurrentTime")
    m.osdTotalDuration = m.top.findNode("osdTotalDuration")
    m.progressBarFill = m.top.findNode("progressBarFill")
    m.osdTimer = m.top.findNode("osdTimer")

    ' Focusable Interactive Pills (6 pills)
    m.pillResume = m.top.findNode("pillResume")
    m.pillResumeLabel = m.top.findNode("pillResumeLabel")
    m.pillSync = m.top.findNode("pillSync")
    m.pillSyncLabel = m.top.findNode("pillSyncLabel")
    m.pillStyle = m.top.findNode("pillStyle")
    m.pillStyleLabel = m.top.findNode("pillStyleLabel")
    m.pillAudio = m.top.findNode("pillAudio")
    m.pillAudioLabel = m.top.findNode("pillAudioLabel")
    m.pillToggle = m.top.findNode("pillToggle")
    m.pillToggleLabel = m.top.findNode("pillToggleLabel")
    m.pillQueue = m.top.findNode("pillQueue")
    m.pillQueueLabel = m.top.findNode("pillQueueLabel")

    ' Observers
    m.video.observeField("state", "onVideoStateChange")
    m.video.observeField("position", "onPositionChange")
    m.subSyncTimer.observeField("fire", "hideSubSyncOSD")
    m.seekToastTimer.observeField("fire", "hideSeekToast")
    m.osdTimer.observeField("fire", "hideOSD")
    m.subtitleTimer.observeField("fire", "updateSubtitleDisplay")

    ' Internal state
    m.cues = []
    m.subOffset = 0.0
    m.subsVisible = true
    m.activePillIndex = 0
    m.numPills = 6
    m.pills = [m.pillResume, m.pillSync, m.pillStyle, m.pillAudio, m.pillToggle, m.pillQueue]
    m.pillLabels = [m.pillResumeLabel, m.pillSyncLabel, m.pillStyleLabel, m.pillAudioLabel, m.pillToggleLabel, m.pillQueueLabel]

    ' Subtitle Styling states
    m.colorIndex = 0
    m.colors = ["0xFFFFFFFF", "0xFBBF24FF", "0x38BDF8FF"]
    m.colorNames = ["White", "Yellow", "Sky Blue"]

    ' Current Video metadata
    m.currentTitle = "UNKNOWN"
    m.callbackUrl = ""

    ' Set initial clean ASCII labels
    if m.pillResumeLabel <> invalid then m.pillResumeLabel.text = "Resume [OK]"
    if m.pillStyleLabel <> invalid then m.pillStyleLabel.text = "Style: White"
    if m.pillAudioLabel <> invalid then m.pillAudioLabel.text = "Audio: Default"
    if m.pillToggleLabel <> invalid then m.pillToggleLabel.text = "Subs: ON"
    if m.pillQueueLabel <> invalid then m.pillQueueLabel.text = "Next Video >>"

    updateSyncLabel()
    updatePillVisuals()
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
    m.osdTitle.text = UCase(m.currentTitle.ToStr())

    if offsetStr <> ""
        m.subOffset = Val(offsetStr)
    else
        m.subOffset = 0.0
    end if
    updateSyncLabel()

    ' Configure and launch video
    m.dashboard.visible = false
    m.video.visible = true

    content = CreateObject("roSGNode", "ContentNode")
    content.url = mediaUrl
    content.title = m.currentTitle

    ' Determine stream format
    lUrl = LCase(mediaUrl)
    if InStr(1, lUrl, ".m3u8") > 0
        content.streamFormat = "hls"
    else if InStr(1, lUrl, ".mpd") > 0
        content.streamFormat = "dash"
    else
        content.streamFormat = "mp4"
    end if

    m.video.content = content
    m.video.control = "play"
    m.subtitleTimer.control = "start"

    ' Fetch subtitles if provided
    if subUrl <> ""
        m.subTask = CreateObject("roSGNode", "SubtitleTask")
        m.subTask.observeField("cues", "onSubtitlesLoaded")
        m.subTask.url = subUrl
        m.subTask.control = "RUN"
    else
        m.cues = []
        m.subtitleGroup.visible = false
    end if

    ' CRITICAL: Keep interactive controls INACTIVE while video is playing
    ' Controls only become active when user presses OK to pause!
    m.osd.visible = false
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
        m.subtitleLabel.text = ""
        m.subtitleShadow.text = ""
        return
    end if

    vidPos = m.video.position + m.subOffset
    currentText = ""

    ' Search for active cue
    for each cue in m.cues
        if vidPos >= cue.start and vidPos <= cue.endPos
            currentText = cue.text
            exit for
        else if vidPos < cue.start
            exit for
        end if
    end for

    if m.subtitleLabel.text <> currentText
        m.subtitleLabel.text = currentText
        m.subtitleShadow.text = currentText
    end if
end sub

sub onPositionChange(event as Object)
    vidPos = event.getData()
    dur = m.video.duration

    if dur > 0
        m.osdCurrentTime.text = formatTime(vidPos)
        m.osdTotalDuration.text = formatTime(dur)

        ' Time Remaining Countdown (-HH:MM:SS)
        remSecs = dur - vidPos
        if remSecs < 0 then remSecs = 0
        m.osdRemaining.text = "-" + formatTime(remSecs)

        ' Progress bar calculation (width out of 1480px)
        ratio = vidPos / dur
        if ratio > 1.0 then ratio = 1.0
        m.progressBarFill.width = Int(ratio * 1480)
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

sub onVideoStateChange(event as Object)
    state = event.getData()
    print "MainScene: video state = " + state

    if state = "finished"
        ' Playback finished, notify backend for playlist auto-advance
        if m.callbackUrl <> ""
            notifyBackend(m.callbackUrl + "?event=finished")
        end if
    else if state = "error"
        m.statusLabel.text = "Playback Error"
        m.statusLabel.color = "0xEF4444FF"
        m.dashboard.visible = true
        m.video.visible = false
        m.subtitleGroup.visible = false
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

    if not m.osd.visible
        ' ==============================================================
        ' CONTROLS ARE NOT ACTIVE (Video is Playing Normally)
        ' Left / Right skips backward / forward.
        ' OK pauses video and brings up the controls deck.
        ' ==============================================================
        if lKey = "ok"
            ' Pause playback and show interactive settings menu
            m.video.control = "pause"
            m.activePillIndex = 0 ' Default focus on Resume [OK]
            updatePillVisuals()
            showOSD()
            return true
        else if lKey = "play"
            togglePlayPause()
            return true
        else if lKey = "left" or lKey = "rewind" or lKey = "rev"
            seekRelative(-15)
            return true
        else if lKey = "right" or lKey = "fastforward" or lKey = "fwd"
            seekRelative(15)
            return true
        else if lKey = "replay" or lKey = "instantreplay"
            seekRelative(-10)
            return true
        else if lKey = "up"
            nudgeSync(0.5)
            return true
        else if lKey = "down"
            nudgeSync(-0.5)
            return true
        else if lKey = "back"
            stopPlayback()
            return true
        end if
    else
        ' ==============================================================
        ' CONTROLS ARE ACTIVE (Video is Paused, Menu is Open)
        ' Left / Right navigates through option pills.
        ' Up / Down adjusts the focused setting.
        ' OK selects / resumes.
        ' Play / Back resumes playback and hides controls.
        ' ==============================================================
        m.osdTimer.control = "stop"
        m.osdTimer.control = "start"

        if lKey = "left"
            ' Navigate pill selection left
            m.activePillIndex = (m.activePillIndex - 1 + m.numPills) Mod m.numPills
            updatePillVisuals()
            return true
        else if lKey = "right"
            ' Navigate pill selection right
            m.activePillIndex = (m.activePillIndex + 1) Mod m.numPills
            updatePillVisuals()
            return true
        else if lKey = "up"
            handlePillAdjust("up")
            return true
        else if lKey = "down"
            handlePillAdjust("down")
            return true
        else if lKey = "ok"
            handlePillOK()
            return true
        else if lKey = "play" or lKey = "back"
            ' Resume playback and dismiss controls
            m.video.control = "resume"
            hideOSD()
            return true
        else if lKey = "rewind" or lKey = "rev"
            seekRelative(-15)
            return true
        else if lKey = "fastforward" or lKey = "fwd"
            seekRelative(15)
            return true
        end if
    end if

    return false
end function

sub handlePillAdjust(direction as String)
    delta = 0.5
    if direction = "down" then delta = -0.5

    if m.activePillIndex = 0
        ' Pill 0: Resume
        m.video.control = "resume"
        hideOSD()
    else if m.activePillIndex = 1
        ' Pill 1: Subtitle Sync Offset
        nudgeSync(delta)
    else if m.activePillIndex = 2
        ' Pill 2: Subtitle Style
        if direction = "up"
            m.colorIndex = (m.colorIndex + 1) Mod m.colors.Count()
        else
            m.colorIndex = (m.colorIndex - 1 + m.colors.Count()) Mod m.colors.Count()
        end if
        m.subtitleLabel.color = m.colors[m.colorIndex]
        m.pillStyleLabel.text = "Style: " + m.colorNames[m.colorIndex]
    else if m.activePillIndex = 3
        ' Pill 3: Audio Track Switcher
        switchAudioTrack()
    else if m.activePillIndex = 4
        ' Pill 4: Subtitles Toggle On/Off
        toggleSubtitles()
    else if m.activePillIndex = 5
        ' Pill 5: Playlist Skip Next
        if m.callbackUrl <> ""
            notifyBackend(m.callbackUrl + "?event=next")
        end if
    end if
end sub

sub handlePillOK()
    if m.activePillIndex = 0
        ' Pill 0: Resume Playback
        m.video.control = "resume"
        hideOSD()
    else if m.activePillIndex = 1
        ' Pill 1: Subtitle Sync - nudge +0.5s on OK
        nudgeSync(0.5)
    else if m.activePillIndex = 2
        ' Pill 2: Subtitle Style - cycle color on OK
        m.colorIndex = (m.colorIndex + 1) Mod m.colors.Count()
        m.subtitleLabel.color = m.colors[m.colorIndex]
        m.pillStyleLabel.text = "Style: " + m.colorNames[m.colorIndex]
    else if m.activePillIndex = 3
        ' Pill 3: Audio Track Switcher
        switchAudioTrack()
    else if m.activePillIndex = 4
        ' Pill 4: Subtitles Toggle On/Off
        toggleSubtitles()
    else if m.activePillIndex = 5
        ' Pill 5: Playlist Skip Next
        if m.callbackUrl <> ""
            notifyBackend(m.callbackUrl + "?event=next")
        end if
    end if
end sub

sub switchAudioTrack()
    tracks = m.video.audioTracks
    if tracks <> invalid and tracks.Count() > 1
        curIndex = m.video.currentAudioTrack
        if curIndex = invalid then curIndex = 0
        nextIndex = (curIndex + 1) Mod tracks.Count()
        m.video.currentAudioTrack = nextIndex
        trackName = "Track " + (nextIndex + 1).ToStr()
        if tracks[nextIndex] <> invalid and tracks[nextIndex].DoesExist("Track")
            trackName = tracks[nextIndex].Track
        end if
        m.pillAudioLabel.text = "Audio: " + trackName
    else
        m.pillAudioLabel.text = "Audio: Default"
    end if
end sub

sub toggleSubtitles()
    m.subsVisible = not m.subsVisible
    m.subtitleGroup.visible = m.subsVisible
    if m.subsVisible
        m.pillToggleLabel.text = "Subs: ON"
    else
        m.pillToggleLabel.text = "Subs: OFF"
    end if
end sub

sub updatePillVisuals()
    for i = 0 to m.numPills - 1
        if i = m.activePillIndex
            m.pills[i].color = "0x38BDF8FF"
            m.pillLabels[i].color = "0x090B10FF"
        else
            m.pills[i].color = "0x1E293BCC"
            m.pillLabels[i].color = "0xE2E8F0FF"
        end if
    end for
end sub

sub nudgeSync(delta as Float)
    m.subOffset = m.subOffset + delta
    updateSyncLabel()
    showSubSyncOSD()
    updateSubtitleDisplay()
end sub

sub updateSyncLabel()
    sign = ""
    if m.subOffset > 0 then sign = "+"
    offsetText = sign + m.subOffset.ToStr() + "s"
    m.subSyncLabel.text = "Sub Sync: " + offsetText
    m.pillSyncLabel.text = "Sync: " + offsetText + " [UP/DN]"
end sub

sub showSubSyncOSD()
    m.subSyncOSD.visible = true
    m.subSyncTimer.control = "stop"
    m.subSyncTimer.control = "start"
end sub

sub hideSubSyncOSD()
    m.subSyncOSD.visible = false
end sub

sub toggleOSD()
    if m.osd.visible
        m.video.control = "resume"
        hideOSD()
    else
        m.video.control = "pause"
        m.activePillIndex = 0
        updatePillVisuals()
        showOSD()
    end if
end sub

sub showOSD()
    m.osd.visible = true
    m.osdTimer.duration = 10
    m.osdTimer.control = "stop"
    m.osdTimer.control = "start"
end sub

sub hideOSD()
    m.osd.visible = false
end sub

sub seekRelative(deltaSeconds as Integer)
    curPos = m.video.position
    if curPos = invalid then curPos = 0.0
    dur = m.video.duration
    newPos = curPos + deltaSeconds
    if newPos < 0 then newPos = 0
    if dur > 0 and newPos > dur then newPos = dur

    m.video.seek = newPos

    ' Show quick feedback toast WITHOUT making the interactive controls deck active
    direction = ">> +"
    if deltaSeconds < 0 then direction = "<< "
    m.seekToastLabel.text = direction + deltaSeconds.ToStr() + "s  [" + formatTime(newPos) + "]"
    m.seekToast.visible = true
    m.seekToastTimer.control = "stop"
    m.seekToastTimer.control = "start"
end sub

sub hideSeekToast()
    m.seekToast.visible = false
end sub

sub togglePlayPause()
    if m.video.state = "playing"
        m.video.control = "pause"
        m.activePillIndex = 0
        updatePillVisuals()
        showOSD()
    else
        m.video.control = "resume"
        hideOSD()
    end if
end sub

sub stopPlayback()
    m.video.control = "stop"
    m.video.visible = false
    m.subtitleGroup.visible = false
    m.dashboard.visible = true
    hideOSD()
end sub
