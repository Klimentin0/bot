box.cfg{
    listen = '0.0.0.0:3301',
    memtx_memory = 1073741824,  
    log_level = 5,
    wal_mode = 'none',          
    background = false,         
    log = 'tarantool.log',      
    worker_pool_threads = 4,    
    net_msg_max = 100000        
}


function ping()
    return box.info().status == 'running' 
        and box.space.votes ~= nil 
        and box.space.counters ~= nil
end

if not box.schema.user.exists('admin') then
    box.schema.user.create('admin', {
        password = 'password',
        if_not_exists = true
    })
    box.schema.user.grant('admin', 'read,write,execute', 'space', 'votes')
    box.schema.user.grant('admin', 'read,write,execute', 'space', 'counters')
    box.schema.user.grant('admin', 'create', 'universe')
end

local function init_schema()
    box.begin()
    
    local votes_format = {
        {name = 'id', type = 'string'},
        {name = 'creator', type = 'string'},
        {name = 'question', type = 'string'},
        {name = 'options', type = 'map'},
        {name = 'status', type = 'string'},
        {name = 'created_at', type = 'datetime'},
        {name = 'ended_at', type = 'datetime', is_nullable = true}
    }
    
    box.schema.space.create('votes', {
        if_not_exists = true,
        format = votes_format,
        engine = 'memtx'
    })
    
    box.space.votes:create_index('primary', {
        parts = {'id'},
        if_not_exists = true,
        unique = true
    })

    box.schema.space.create('counters', {
        if_not_exists = true,
        format = {
            {name = 'key', type = 'string'},
            {name = 'value', type = 'unsigned'}
        },
        engine = 'memtx'
    })
    
    box.space.counters:create_index('primary', {
        parts = {'key'},
        if_not_exists = true,
        unique = true
    })

    if not box.space.counters:get('vote_id') then
        box.space.counters:insert{'vote_id', 0}
    end
    
    box.commit()
end

local max_retries = 5
local base_delay = 1.0  
local current_delay = base_delay

for attempt = 1, max_retries do
    local ok, err = pcall(init_schema)
    
    if ok then
        box.info("Schema initialized successfully")
        break
    else
        box.error("Schema initialization attempt %d failed: %s", attempt, err)
        
        if attempt == max_retries then
            error("Fatal: Failed to initialize schema after "..max_retries.." attempts")
        end
        
        local jitter = math.random() * 0.5  
        require('fiber').sleep(current_delay * (1 + jitter))
        current_delay = current_delay * 2
    end
end

box.space.votes:create_index('status_idx', {
    parts = {'status'},
    if_not_exists = true,
    unique = false
})

box.info("Tarantool instance ready to accept connections")