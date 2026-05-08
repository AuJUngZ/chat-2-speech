export namespace config {
	
	export class Position {
	    x: number;
	    y: number;
	
	    static createFrom(source: any = {}) {
	        return new Position(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.x = source["x"];
	        this.y = source["y"];
	    }
	}
	export class Config {
	    _comment: string;
	    autoFadeDelay: number;
	    cloudTTSAPIKey: string;
	    cloudTTSEnabled: boolean;
	    englishVoiceName: string;
	    maxQueueSize: number;
	    overlayPosition: Position;
	    pinLastMessageHotkey: string;
	    speechRateMultiplier: number;
	    thaiVoiceName: string;
	    toggleOverlayHotkey: string;
	    twitchChannel: string;
	    twitchOAuthToken: string;
	    ttsEngine: string;
	    geminiVoiceName: string;
	    geminiModel: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this._comment = source["_comment"];
	        this.autoFadeDelay = source["autoFadeDelay"];
	        this.cloudTTSAPIKey = source["cloudTTSAPIKey"];
	        this.cloudTTSEnabled = source["cloudTTSEnabled"];
	        this.englishVoiceName = source["englishVoiceName"];
	        this.maxQueueSize = source["maxQueueSize"];
	        this.overlayPosition = this.convertValues(source["overlayPosition"], Position);
	        this.pinLastMessageHotkey = source["pinLastMessageHotkey"];
	        this.speechRateMultiplier = source["speechRateMultiplier"];
	        this.thaiVoiceName = source["thaiVoiceName"];
	        this.toggleOverlayHotkey = source["toggleOverlayHotkey"];
	        this.twitchChannel = source["twitchChannel"];
	        this.twitchOAuthToken = source["twitchOAuthToken"];
	        this.ttsEngine = source["ttsEngine"];
	        this.geminiVoiceName = source["geminiVoiceName"];
	        this.geminiModel = source["geminiModel"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace irc {
	
	export class ChatMessage {
	    Username: string;
	    Text: string;
	    Channel: string;
	    Raw: string;
	    Platform: string;
	
	    static createFrom(source: any = {}) {
	        return new ChatMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Username = source["Username"];
	        this.Text = source["Text"];
	        this.Channel = source["Channel"];
	        this.Raw = source["Raw"];
	        this.Platform = source["Platform"];
	    }
	}

}

export namespace main {
	
	export class TTSInfo {
	    engine: string;
	    thaiVoices: string[];
	    englishVoices: string[];
	    geminiVoices: string[];
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new TTSInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.engine = source["engine"];
	        this.thaiVoices = source["thaiVoices"];
	        this.englishVoices = source["englishVoices"];
	        this.geminiVoices = source["geminiVoices"];
	        this.error = source["error"];
	    }
	}

}

