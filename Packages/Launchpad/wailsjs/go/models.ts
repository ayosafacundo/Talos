export namespace main {
	
	export class AppManifestView {
	    id: string;
	    name: string;
	    icon: string;
	    url: string;
	    description: string;
	    category: string;
	    store_url?: string;
	    allowed_origins?: string[];
	    development?: boolean;
	    trust_status?: string;
	
	    static createFrom(source: any = {}) {
	        return new AppManifestView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.icon = source["icon"];
	        this.url = source["url"];
	        this.description = source["description"];
	        this.category = source["category"];
	        this.store_url = source["store_url"];
	        this.allowed_origins = source["allowed_origins"];
	        this.development = source["development"];
	        this.trust_status = source["trust_status"];
	    }
	}
	export class PackageDirDevRow {
	    dir_name: string;
	    has_manifest: boolean;
	    dev_mode: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PackageDirDevRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.dir_name = source["dir_name"];
	        this.has_manifest = source["has_manifest"];
	        this.dev_mode = source["dev_mode"];
	    }
	}
	export class PackageLocalHTTPResponse {
	    status: number;
	    content_type: string;
	    body: string;
	    body_base64?: string;
	
	    static createFrom(source: any = {}) {
	        return new PackageLocalHTTPResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.content_type = source["content_type"];
	        this.body = source["body"];
	        this.body_base64 = source["body_base64"];
	    }
	}
	export class PermissionAuditEntry {
	    ts: string;
	    action: string;
	    app_id: string;
	    scope: string;
	    granted: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new PermissionAuditEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ts = source["ts"];
	        this.action = source["action"];
	        this.app_id = source["app_id"];
	        this.scope = source["scope"];
	        this.granted = source["granted"];
	        this.message = source["message"];
	    }
	}
	export class PermissionEntry {
	    app_id: string;
	    scope: string;
	    granted: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PermissionEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.app_id = source["app_id"];
	        this.scope = source["scope"];
	        this.granted = source["granted"];
	    }
	}
	export class RemotePackageDescriptor {
	    id: string;
	    name: string;
	    source?: string;
	    install_url?: string;
	
	    static createFrom(source: any = {}) {
	        return new RemotePackageDescriptor(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.source = source["source"];
	        this.install_url = source["install_url"];
	    }
	}
	export class ThemeInfo {
	    name: string;
	    file: string;
	
	    static createFrom(source: any = {}) {
	        return new ThemeInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.file = source["file"];
	    }
	}
	export class UpdateEntryView {
	    app_id: string;
	    version: string;
	    artifact_url: string;
	    min_host_version?: string;
	    signature_url?: string;
	    name?: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateEntryView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.app_id = source["app_id"];
	        this.version = source["version"];
	        this.artifact_url = source["artifact_url"];
	        this.min_host_version = source["min_host_version"];
	        this.signature_url = source["signature_url"];
	        this.name = source["name"];
	    }
	}
	export class UserPrefs {
	    theme: string;
	    tab_colors: Record<string, string>;
	    developer_mode?: boolean;
	    dev_mode_by_dir?: Record<string, boolean>;
	    dev_prefs_migrated?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UserPrefs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.tab_colors = source["tab_colors"];
	        this.developer_mode = source["developer_mode"];
	        this.dev_mode_by_dir = source["dev_mode_by_dir"];
	        this.dev_prefs_migrated = source["dev_prefs_migrated"];
	    }
	}

}

