function scrape(config)
    local max_items = config.max_items or 10
    
    local response, err = http.get("https://techcrunch.com")
    if err then
        log.error("Failed to fetch TechCrunch: " .. err)
        return {}
    end
    
    if response.status ~= 200 then
        log.error("TechCrunch returned status: " .. response.status)
        return {}
    end
    
    local doc, err = html.parse(response.body)
    if err then
        log.error("Failed to parse HTML: " .. err)
        return {}
    end
    
    local articles = html.select(doc, ".loop-card")
    local items = {}
    local count = 0
    
    for i = 1, #articles do
        if count >= max_items then
            break
        end
        
        local article = articles[i]
        
        local title_elem = html.select_one(article, ".loop-card__title a")
        local link_elem = html.select_one(article, ".loop-card__title a")
        local author_elem = html.select_one(article, ".loop-card__meta a")
        local time_elem = html.select_one(article, "time")
        local img_elem = html.select_one(article, "img")
        
        if title_elem and link_elem then
            local title = html.text(title_elem)
            local url = html.attr(link_elem, "href")
            local author = ""
            local published = ""
            local thumbnail = ""
            
            if author_elem then
                author = html.text(author_elem)
            end
            
            if time_elem then
                published = html.attr(time_elem, "datetime") or html.text(time_elem)
            end
            
            if img_elem then
                thumbnail = html.attr(img_elem, "src")
            end
            
            if url and url ~= "" and title and title ~= "" then
                local item = {
                    id = url,
                    title = title,
                    url = url,
                    author = author,
                    published = published,
                    thumbnail = thumbnail,
                    content = "",
                    metadata = {
                        source = "techcrunch",
                        scraped = true
                    }
                }
                
                table.insert(items, item)
                count = count + 1
            end
        end
    end
    
    log.info("Scraped " .. count .. " articles from TechCrunch")
    
    return items
end
