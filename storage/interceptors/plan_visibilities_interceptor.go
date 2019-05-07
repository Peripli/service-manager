package interceptors

// when a plan is updated from public to private or vise versa this is broker interceptor
// for each catalog plan check if
// resync visibilities with cis desired state
// its changed and resync visibilities

// if we add CF in supported platforms this triggers broker update which resyncs visibilities with cis
// visibilities that already exist in smDB but for which the notification was previously ignored needs to be processed now

//variant 1 -  update plan interceptor that generates visibility notifications if the visibility already exists
// checks the current supported platforms and the updated one
// makes a diff (for each desired remove it from existing, if not found create ADDED notifiation)
// and for the  ones left in existing, create DELETED notification
// has to read a lot of visibilities from smdb

//variant 2 - resync visibilities s cisa da vzima pod vnimanie supported platforms i da skipva visibilita na bazata na supported platforms
// entitlements api-to trqbva da pravi sustoto
// oss public interceptora trqbva da pravi sustoto - ??
// bi bilo po vuzmojno ako suzdavaneto na vizibilita se premesti kato interceptor na create/update/delete plan

//variant 3 - vizibilito izobsto da ne se suzdava
// create vis interceptor koito sravnqva tipa na platformata s id v vis s supportedplatforms na plana s id v vis

// supported_platforms
// 1. get brokers, get services, get plans, filters to filter out the respective resources when the proxy requests them so that no "unknowns" are returned from the API
// 2. get catalog filter to filter out the catalog when the platform requests it
// 3. create vis interceptor to filter out visibilities that should not be created at all due to supported platforms restrictions - if public vis - fetch all platforms..

// ako brokera (notification) e bil izcqlo filteriran predi tova 6toto cataloga my za tazi platforma e osaval prazen
// kato se update-ne brokera s modnat plan (supportedplatforms +cf) shte se generira update broker notifikaciq za tazi platforma bez da ima broker tam
