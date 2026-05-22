export namespace app {
	
	export class ImageInfo {
	    index: number;
	    name: string;
	    dataUrl: string;
	    mimeType: string;
	    width: number;
	    height: number;
	
	    static createFrom(source: any = {}) {
	        return new ImageInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.index = source["index"];
	        this.name = source["name"];
	        this.dataUrl = source["dataUrl"];
	        this.mimeType = source["mimeType"];
	        this.width = source["width"];
	        this.height = source["height"];
	    }
	}
	export class VolumeInfo {
	    name: string;
	    path: string;
	    isZip: boolean;
	    isDir: boolean;
	    images: number;
	    hasSubdirectories: boolean;
	
	    static createFrom(source: any = {}) {
	        return new VolumeInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.isZip = source["isZip"];
	        this.isDir = source["isDir"];
	        this.images = source["images"];
	        this.hasSubdirectories = source["hasSubdirectories"];
	    }
	}

}

