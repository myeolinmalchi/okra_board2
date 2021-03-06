package services

import (
    "context"
	"fmt"
	"log"
	"okra_board2/config"
	"okra_board2/models"
	"okra_board2/repositories"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/PuerkitoBio/goquery"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"golang.org/x/net/html"
	"gorm.io/gorm"
)

type PostService interface {

    // 게시물을 작성하고 postId와 유효성 검사 결과 및 에러를 반환한다.
    // post.Thumbnail이 비어있을 경우 "default_thumbnail.png"로 설정한다.
    WritePost(post *models.Post)    (postId int, result *models.PostValidationResult, err error)

    // 게시물을 업데이트하고 유효성 검사 결과와 에러를 반환한다.
    // post.Thumbnail이 비어있을 경우 "default_thumbnail.png"로 설정한다.
    UpdatePost(post *models.Post)   (result *models.PostValidationResult, err error)

    // 게시물을 삭제하고 에러를 반환한다.
    // 게시물에 포함된 이미지도 함께 삭제한다.
    DeletePost(postId int)          (err error)

    // 게시글을 불러온다.
    // enabled 속성이 true일 경우, 
    // status 열이 false인 게시물에 대하여 
    // RecordNotFound 에러를 반환한다.
    GetPost(status *bool, postId int)             (post *models.Post, err error)
    
    // 조건에 부합하는 게시글의 개수와 함께 게시글 배열을 반환한다.
    // enabled 속성이 true일 경우, status 열이 true인 게시글만을 불러온다.
    // page, size는 페이지네이션을 위한 속성이다.
    // boardId 속성이 nil일 경우 전체 게시판에서 게시글을 검색한다.
    // keyword 속성이 nil이 아닐 경우 제목에 keyword가 포함된 게시글만을 검색한다.
    GetPosts(
        status bool,
        selected *bool,
        page, size int,
        boardId *int,
        titleKeyword *string,
        tagKeyword *string,
    )                               (posts []models.Post, count int)

    // selected colunm이 true인 게시글들의 썸네일 및 제목 정보를 불러온다.
    GetSelectedThumbnails()         (thumbnaiils []models.Thumbnail)

    // selected column이 true인 게시물을 재설정한다.
    // 전달받은 id 목록 중 존재하지 않는 게시물이 있을 경우
    // 해당 id 리스트를 gorm.ErrRecordNotFound와 함께 반환한다.
    ResetSelectedPosts(ids *[]int)  ([]int, error)

    // 사용되지 않는 이미지를 s3에서 삭제한다.
    DeleteUnusedImages()            (err error)

}

type PostServiceImpl struct {
    postRepo    repositories.PostRepository
    conf        *config.Config
    client      *s3.Client
}

func NewPostServiceImpl(
    postRepo repositories.PostRepository,
    conf *config.Config,
    client *s3.Client,
) PostService {
    return &PostServiceImpl{
        postRepo: postRepo,
        conf: conf,
        client: client,
    }
}

func (r *PostServiceImpl) checkContent(content string) *string {
    var msg string
    if content == "" || content == "<p><br></p>"{
        msg = "내용을 입력하세요."
    } else {
        return nil
    }
    return &msg
}

func (r *PostServiceImpl) checkTitle(title string) *string {
    var msg string 
    if title == "" {
        msg  = "제목을 입력하세요."
    } else {
        return nil
    }
    return &msg
}

func (r *PostServiceImpl) checkThumbnail(thumbnail string) *string {
    var msg string 
    if thumbnail == "" || thumbnail == "<p><br></p>" {
        msg = ""
    } else {
        return nil
    }
    return &msg
}

func (r *PostServiceImpl) postValidation(post *models.Post) *models.PostValidationResult {
    if thumbnailCheck := r.checkThumbnail(post.Thumbnail); thumbnailCheck != nil {
        post.Thumbnail = fmt.Sprintf(
            `<p><img src="https://%s/images/%s"/></p>`,
            r.conf.AWS.Domain,
            os.Getenv("DEFAULT_THUMBNAIL"),
        )
        fmt.Println(post.Thumbnail)
    }
    result := &models.PostValidationResult {
        Title: r.checkTitle(post.Title),
        Content: r.checkContent(post.Content),
    }
    return result.GetOrNil()
}

// 태그 중복 제거
func (r *PostServiceImpl) distinctTags(t []models.PostTag) []models.PostTag {
    keys := make(map[string]struct{}) 
    res := make([]models.PostTag, 0) 
    for _, val := range t { 
        if _, ok := keys[val.Name]; ok { 
            continue 
        } else { 
            keys[val.Name] = struct{}{} 
            res = append(res, val) 
        } 
    }
    return res 
}

func (r *PostServiceImpl) WritePost(post *models.Post) (postId int, result *models.PostValidationResult,  err error) {
    result = r.postValidation(post)
    if result == nil {
        post.Tags = r.distinctTags(post.Tags)
        postId, err = r.postRepo.InsertPost(post)
    }
    return
}

func (r *PostServiceImpl) UpdatePost(post *models.Post) (result *models.PostValidationResult, err error) {
    result = r.postValidation(post)
    if result == nil {
        err = r.postRepo.UpdatePost(post)
    }
    return
}

func (r *PostServiceImpl) deleteImageFromHTML(htmlStr string) (err error) {
    node, err := html.Parse(strings.NewReader(htmlStr))
    if err != nil { return }

    doc := goquery.NewDocumentFromNode(node)

    images := doc.Find("img")
    images.Each(func(idx int, img *goquery.Selection) {
        //"https://~~~.com/public.okraseoul.com/images/filename"
        src := img.AttrOr("src", "")
        temp := strings.Split(src, "/")
        filename := temp[5]
        if filename == os.Getenv("DEFAULT_THUMBNAIL") {
            return
        }
        _, err := r.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput {
            Bucket: aws.String(r.conf.AWS.Bucket),
            Key:    aws.String("images/"+filename),
        })
        if err!= nil {
            log.Println(err)
        } else {
            log.Println("이미지가 삭제되었습니다: "+filename)
        }
    })
    return nil
}

func (r *PostServiceImpl) DeleteUnusedImages() (err error) {
    //posts := r.postRepo.GetAllPosts();
    params := &s3.ListObjectsInput {
        Bucket: aws.String(r.conf.AWS.Bucket),
        Prefix: aws.String("images"),
    }
    resp, err := r.client.ListObjects(context.TODO(), params)
    if err != nil  { return }

    var filenames []string

    for _, key := range resp.Contents {
        filenames = append(filenames, *key.Key)
    }

    return
}

func (r *PostServiceImpl) DeletePost(postId int) (err error) {
    post, err := r.postRepo.GetPost(nil, postId)
    if err != nil { return }

    err = r.deleteImageFromHTML(post.Content)
    if err != nil { return }

    err = r.deleteImageFromHTML(post.Thumbnail)
    if err != nil { return }

    return r.postRepo.DeletePost(postId)
}

func (r *PostServiceImpl) GetPost(status *bool, postId int) (post *models.Post, err error) {
    post, err = r.postRepo.GetPost(status, postId)
    if err != nil { return }

    if prevPost, err := r.postRepo.GetPrevPostInfo(status, postId); err == nil {
        post.Prev = prevPost
    }

    if nextPost, err := r.postRepo.GetNextPostInfo(status, postId); err == nil {
        post.Next = nextPost
    }

    return
}

func (r *PostServiceImpl) GetPosts(
    enabled bool,
    selected *bool,
    page, size int,
    boardId *int,
    titleKeyword *string,
    tagKeyword *string,
) (posts []models.Post, count int) {
    posts, count = r.postRepo.GetPosts(
        enabled,
        selected,
        page, size,
        boardId, 
        titleKeyword, 
        tagKeyword,
        "post_id desc",
    )
    return
}

func (r *PostServiceImpl) GetSelectedThumbnails() (thumbnails []models.Thumbnail){
    return r.postRepo.GetSelectedThumbnails()
}

func (r *PostServiceImpl) ResetSelectedPosts(ids *[]int) ([]int, error) {
    var nonexistids []int
    for _, id := range *ids {
        if !r.postRepo.CheckPostExists(id) {
            nonexistids = append(nonexistids, id)
        }
    }
    if len(nonexistids) > 0 {
        return nonexistids, gorm.ErrRecordNotFound
    }
    return nil, r.postRepo.ResetSelectedPost(ids)
}
