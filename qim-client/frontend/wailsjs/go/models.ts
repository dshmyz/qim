export namespace main {
	
	export class AppInfo {
	    version: string;
	    platform: string;
	    userDataDir: string;
	
	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.platform = source["platform"];
	        this.userDataDir = source["userDataDir"];
	    }
	}
	export class FileDialogOptions {
	    title: string;
	    defaultDir: string;
	    filters: string[];
	
	    static createFrom(source: any = {}) {
	        return new FileDialogOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.defaultDir = source["defaultDir"];
	        this.filters = source["filters"];
	    }
	}
	export class FileDialogResult {
	    canceled: boolean;
	    filePath: string;
	
	    static createFrom(source: any = {}) {
	        return new FileDialogResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.canceled = source["canceled"];
	        this.filePath = source["filePath"];
	    }
	}
	export class ScreenSource {
	    id: string;
	    name: string;
	    thumbnail: string;
	
	    static createFrom(source: any = {}) {
	        return new ScreenSource(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.thumbnail = source["thumbnail"];
	    }
	}
	export class UpdateInfo {
	    available: boolean;
	    version: string;
	    url: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.version = source["version"];
	        this.url = source["url"];
	    }
	}

}

